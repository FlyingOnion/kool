package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	NewLine       = "\n"
	DoubleNewLine = "\n\n"
)

func retrieveImports(file *ast.File) sets.Set[string] {
	imports := sets.New[string]()
	for _, imp := range file.Imports {
		imports.Insert(imp.Path.Value)
	}
	return imports
}

// retrieveControllerMethods returns the methods of the controller in file custom.go
//
// controllerName is the name of the controller, by default "c", such as
//
//	func(--->c *Controller)
func retrieveControllerMethods(file *ast.File, controllerName string) sets.Set[string] {
	methods := sets.New[string]()
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}
		starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		ident, ok := starExpr.X.(*ast.Ident)
		if !ok || ident.Name != controllerName {
			continue
		}
		methods.Insert(funcDecl.Name.Name)
	}
	return methods
}

func createOrRewriteGoMod(goModTmpl *template.Template, config *Controller) {
	fp := filepath.Join(config.Base, "go.mod")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		f, err := os.Create(fp)
		if err != nil {
			log.Fatal("failed to write file", "file", fp, "cause", err)
		}
		err = goModTmpl.Execute(f, config)
		f.Close()
		if err != nil {
			log.Fatal("failed to execute template", "template", goModTmpl.Name(), "cause", err)
		}
		return
	}
}

func createOrUpdateCustom(customTmpl *template.Template, config *Controller) {
	fp := filepath.Join(config.Base, customTmpl.Name()+".go")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		f, err := os.Create(fp)
		if err != nil {
			log.Fatal("failed to write file", "file", fp, "cause", err)
		}
		err = customTmpl.Execute(f, config)
		f.Close()
		if err != nil {
			log.Fatal("failed to execute template", "template", customTmpl.Name(), "cause", err)
		}
		return
	}

	log.Debug(fp + " already exists")
	log.Info("try to add new codes ...")
	log.Info("(koolbuilder will not rewrite existing codes)")

	f1, err := os.Open(fp)
	if err != nil {
		log.Fatal("failed to open file", "file", fp, "cause", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(f1)
	b1 := make([]byte, buf.Len())
	copy(b1, buf.Bytes())
	f1.Close()

	fset := token.NewFileSet()
	target, err := parser.ParseFile(fset, "", b1, parser.AllErrors|parser.ParseComments)
	if err != nil {
		log.Fatal("failed to parse AST from existing file", "file", fp, "cause", err)
	}

	buf.Reset()
	customTmpl.Execute(&buf, config)
	b2 := make([]byte, buf.Len())
	copy(b2, buf.Bytes())
	cur, err := parser.ParseFile(token.NewFileSet(), "", b2, parser.AllErrors|parser.ParseComments)
	if err != nil {
		log.Fatal("failed to parse AST from template", "template", customTmpl.Name(), "cause", err)
	}

	// try to add missing imports
	var g *ast.GenDecl
	var hasImport bool
	for _, decl := range target.Decls {
		if g, hasImport = decl.(*ast.GenDecl); hasImport && g.Tok == token.IMPORT {
			break
		}
	}

	// if there's no import declaration, create one
	if g == nil {
		log.Debug("no import declaration found in existing file", "file", fp)
		g = &ast.GenDecl{
			Tok: token.IMPORT,
		}
	}

	existedImports := retrieveImports(target)
	for _, imp := range cur.Imports {
		if existedImports.Has(imp.Path.Value) {
			continue
		}
		log.Debug("new package will be added to import list", "package", imp.Path.Value)

		g.Specs = append(g.Specs, imp)
	}
	if !hasImport && len(g.Specs) > 0 {
		log.Debug("creating import declaration")
		target.Decls = append([]ast.Decl{g}, target.Decls...)
	}

	// write to a temporary buffer, so we can add methods
	var tmpBuf bytes.Buffer
	printer.Fprint(&tmpBuf, fset, &printer.CommentedNode{
		Node:     target,
		Comments: target.Comments,
	})

	existedMethods := retrieveControllerMethods(target, config.Name)
	for _, decl := range cur.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}
		starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		ident, ok := starExpr.X.(*ast.Ident)
		if !ok || ident.Name != config.Name || existedMethods.Has(funcDecl.Name.Name) {
			continue
		}

		log.Debug("new method will be added",
			"receiver", "*"+config.Name,
			"method", funcDecl.Name.Name,
		)
		tmpBuf.WriteString(NewLine)
		tmpBuf.Write(b2[funcDecl.Pos()-1 : funcDecl.End()-1])
		tmpBuf.WriteString(NewLine)
	}
	if tmpBuf.Len() == len(b1) {
		log.Debug("no new codes")
	}

	f1, err = os.Create(fp)
	if err != nil {
		log.Fatal("failed to create file", "file", fp, "cause", err)
	}
	log.Debug("writing file", "file", fp)
	tmpBuf.WriteTo(f1)
	f1.Close()
}

func createOrRewrite(tmpl *template.Template, config *Controller) {
	fp := filepath.Join(config.Base, tmpl.Name()+".go")
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal("failed to write file", "file", fp, "cause", err)
	}
	err = tmpl.Execute(f, config)
	f.Close()
	if err != nil {
		log.Fatal("failed to execute template", "template", tmpl.Name(), "cause", err)
	}
}

func createOrRewriteDeepCopy(tmpl *template.Template, config *Controller) {
	for i := range config.Resources {
		if !config.Resources[i].GenDeepCopy {
			continue
		}

		var fp string
		if len(config.Resources[i].Package) == 0 {
			fp = filepath.Join(config.Base, config.Resources[i].LowerKind+"_gen.deepcopy.go")
		} else {
			// filepath = basedir + (package - gomodule)
			relativePath, err := filepath.Rel(config.Go.Module, config.Resources[i].Package)
			if err != nil {
				log.Fatal("failed to get relative path", "module", config.Go.Module, "package", config.Resources[i].Package, "cause", err)
			}
			fp = filepath.Join(config.Base, relativePath, config.Resources[i].LowerKind+"_gen.deepcopy.go")
		}
		log.Info("write deepcopy of", config.Resources[i].Kind, "to", fp)

		f, err := os.Create(fp)
		if err != nil {
			log.Fatal("failed to create file", "file", fp, "cause", err)
		}
		tmpl.Execute(f, &(config.Resources[i]))
		f.Close()
	}
}

func readConfig(filepath string) *Controller {
	yamlFile, err := os.Open(filepath)
	if err != nil {
		log.Fatal("failed to read file", "file", filepath, "cause", err)
	}
	config := defaultController()
	err = yaml.NewDecoder(yamlFile).Decode(config)
	yamlFile.Close()
	if err != nil {
		log.Fatal("failed to parse config", "cause", err)
	}
	return config
}

func main() {
	config := readConfig("koolpod-controller.yaml")
	config.initAndValidate()

	log.Info("initializing go.mod")
	createOrRewriteGoMod(tmplGoMod, config)

	log.Info("initializing main files")
	createOrRewrite(tmplMain, config)

	log.Debug("initialzing controller.go")
	createOrRewrite(tmplController, config)

	log.Info("initializing custom methods")
	createOrUpdateCustom(tmplCustom, config)

	log.Info("generating deepcopy methods")
	createOrRewriteDeepCopy(tmplDeepCopy, config)
	log.Info("done")

	log.Info("run go mod tidy")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = config.Base
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal("failed to run go mod tidy", "cause", err)
	}
}
