package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/FlyingOnion/pkg/log"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	NewLine = "\n"
	Tab     = "\t"
)

func mustGetOrFatal[T any](t T, err error) T {
	if err != nil {
		os.Exit(1)
	}
	return t
}

func mustHaveNoError(err error) {
	if err != nil {
		os.Exit(1)
	}
}

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

func createOrRewriteGoMod(goModTmpl *template.Template, config *Controller) (err error) {
	log.Info("initializing go.mod")
	fp := filepath.Join(config.Base, "go.mod")
	_, err = os.Stat(fp)
	if !os.IsNotExist(err) {
		return
	}
	f, err := os.Create(fp)
	if err != nil {
		log.Error("failed to write file", "file", fp, "cause", err)
		return
	}
	defer f.Close()
	err = goModTmpl.Execute(f, config)
	if err != nil {
		log.Error("failed to execute template", "template", goModTmpl.Name(), "cause", err)
	}
	return
}

func createOrUpdateCustom(customTmpl *template.Template, config *Controller) (err error) {
	fp := filepath.Join(config.Base, customTmpl.Name()+".go")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return createOrRewrite(customTmpl, config)
	}
	log.Info("update file", "file", customTmpl.Name()+".go")
	f1, err := os.Open(fp)
	if err != nil {
		log.Error("failed to open file", "file", fp, "cause", err)
		return
	}

	var buf bytes.Buffer
	buf.ReadFrom(f1)
	b1 := make([]byte, buf.Len())
	copy(b1, buf.Bytes())
	f1.Close()

	fset := token.NewFileSet()
	target, err := parser.ParseFile(fset, "", b1, parser.AllErrors|parser.ParseComments)
	if err != nil {
		log.Error("failed to parse AST from existing file", "file", fp, "cause", err)
		return
	}

	buf.Reset()
	customTmpl.Execute(&buf, config)
	b2 := make([]byte, buf.Len())
	copy(b2, buf.Bytes())
	cur, err := parser.ParseFile(token.NewFileSet(), "", b2, parser.AllErrors|parser.ParseComments)
	if err != nil {
		log.Error("failed to parse AST from new template", "template", customTmpl.Name(), "cause", err)
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
		log.Info("no import declaration found in existing file", "file", fp)
		g = &ast.GenDecl{
			Tok: token.IMPORT,
		}
	}

	existedImports := retrieveImports(target)
	for _, imp := range cur.Imports {
		if existedImports.Has(imp.Path.Value) {
			continue
		}
		log.Info("add new package to import list", "package", imp.Path.Value)

		g.Specs = append(g.Specs, imp)
	}
	if !hasImport && len(g.Specs) > 0 {
		log.Info("create import block")
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

		log.Info("add new method", "method", "(*"+config.Name+")"+funcDecl.Name.Name)
		tmpBuf.WriteString(NewLine)
		tmpBuf.Write(b2[funcDecl.Pos()-1 : funcDecl.End()-1])
		tmpBuf.WriteString(NewLine)
	}
	if tmpBuf.Len() == len(b1) {
		log.Info("no new code")
		return
	}

	f1, err = os.Create(fp)
	if err != nil {
		log.Error("failed to create file", "file", fp, "cause", err)
	}
	defer f1.Close()
	log.Info("write to file", "file", fp)
	tmpBuf.WriteTo(f1)
	return
}

func createOrRewrite(tmpl *template.Template, config *Controller) (err error) {
	log.Info("create or rewrite file", "file", tmpl.Name()+".go")
	fp := filepath.Join(config.Base, tmpl.Name()+".go")
	f, err := os.Create(fp)
	if err != nil {
		log.Error("failed to write file", "file", fp, "cause", err)
		return
	}
	defer f.Close()
	err = tmpl.Execute(f, config)
	if err != nil {
		log.Error("failed to execute template", "template", tmpl.Name(), "cause", err)
	}
	return
}

func createOrRewriteDeepCopy(tmpl *template.Template, config *Controller) error {
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
				log.Error("failed to get relative path", "module", config.Go.Module, "package", config.Resources[i].Package, "cause", err)
				return err
			}
			fp = filepath.Join(config.Base, relativePath, config.Resources[i].LowerKind+"_gen.deepcopy.go")
		}
		log.Info("write deepcopy", "resource", config.Resources[i].Kind, "file", fp)

		f, err := os.Create(fp)
		if err != nil {
			log.Error("failed to create file", "file", fp, "cause", err)
			return err
		}
		tmpl.Execute(f, &(config.Resources[i]))
		f.Close()
	}
	return nil
}

func readConfig(filepath string) (*Controller, error) {
	if strings.HasPrefix(filepath, "http://") || strings.HasPrefix(filepath, "https://") {
		log.Info("fetching config file", "file", filepath)
		resp, err := http.Get(filepath)
		if err != nil {
			log.Error("failed to fetch file", "file", filepath, "cause", err)
			return nil, err
		}
		defer resp.Body.Close()
		return readConfigFromReader(resp.Body)
	}
	log.Info("read config file", "file", filepath)
	yamlFile, err := os.Open(filepath)
	if err != nil {
		log.Error("failed to open file", "file", filepath, "cause", err)
		return nil, err
	}
	defer yamlFile.Close()
	return readConfigFromReader(yamlFile)
}

func readConfigFromReader(reader io.Reader) (*Controller, error) {
	config := defaultController()
	err := yaml.NewDecoder(reader).Decode(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func runGoModTidy(config *Controller) error {
	log.Info("run go mod tidy")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = config.Base
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Error("failed to run go mod tidy", "cause", err)
	}
	return err
}

func main() {
	var configFile string
	pflag.StringVarP(&configFile, "filename", "f", "", "configuration file of the operator")
	pflag.Parse()

	if len(configFile) == 0 {
		log.Error("missing configuration file")
		log.Info("usage: koolbuilder -f config.yaml")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	config := mustGetOrFatal(readConfig(configFile))
	config.initAndValidate()

	mustHaveNoError(createOrRewriteGoMod(tmplGoMod, config))
	mustHaveNoError(createOrRewrite(tmplMain, config))
	mustHaveNoError(createOrRewrite(tmplController, config))
	mustHaveNoError(createOrUpdateCustom(tmplCustom, config))
	mustHaveNoError(createOrRewriteDeepCopy(tmplDeepCopy, config))
	mustHaveNoError(runGoModTidy(config))
	log.Info("all done")
}
