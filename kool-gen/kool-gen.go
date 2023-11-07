package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
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
//	func(c *Controller)
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

func main() {
	yamlFile, err := os.Open("koolpod-controller.yaml")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	config := Controller{
		// Enqueue: "ratelimiting",
		Retry: 3,
	}
	yaml.NewDecoder(yamlFile).Decode(&config)
	yamlFile.Close()
	if err = config.initAndValidate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tmpl := template.New("base").Funcs(sprig.FuncMap())
	d, err := os.ReadDir("../tmpl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, dirEntry := range d {
		name := dirEntry.Name()
		if !strings.HasSuffix(name, "go.tmpl") {
			continue
		}
		fileName := strings.TrimSuffix(name, ".tmpl")
		fmt.Printf("initializing %s from template\n", fileName)
		tmpl2, err := tmpl.New(name).ParseFiles("../tmpl/" + name)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		f, err := os.Create("./gen/" + fileName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		tmpl2.Execute(f, config)
		f.Close()
	}
	fmt.Println("initializing custom methods")
	customTmpl, err := tmpl.New("custom").ParseFiles("../tmpl/custom")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err = os.Stat("./gen/custom.go"); os.IsNotExist(err) {
		f, err := os.Create("./gen/custom.go")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		customTmpl.Execute(f, config)
		f.Close()
		fmt.Println("done")
		return
	}

	fmt.Println("custom.go already exists, try to add new codes ...")
	fmt.Println("(kool-gen will not rewrite existing codes)")

	f1, err := os.Open("./gen/custom.go")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	buf.ReadFrom(f1)
	b1 := make([]byte, buf.Len())
	copy(b1, buf.Bytes())
	f1.Close()

	fset := token.NewFileSet()
	target, err := parser.ParseFile(fset, "", b1, parser.AllErrors|parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	buf.Reset()
	customTmpl.Execute(&buf, config)
	b2 := make([]byte, buf.Len())
	copy(b2, buf.Bytes())
	cur, err := parser.ParseFile(token.NewFileSet(), "", b2, parser.AllErrors|parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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
		fmt.Println("no import declaration found in file")
		g = &ast.GenDecl{
			Tok: token.IMPORT,
		}
	}

	existedImports := retrieveImports(target)
	for _, imp := range cur.Imports {
		if existedImports.Has(imp.Path.Value) {
			continue
		}
		fmt.Println("new import will be added:", imp.Path.Value)

		g.Specs = append(g.Specs, imp)
	}
	if !hasImport && len(g.Specs) > 0 {
		fmt.Println("creating import declaration")
		target.Decls = append([]ast.Decl{g}, target.Decls...)
	}

	// write to a temporary buffer, so we can add methods
	var tmpBuf bytes.Buffer
	printer.Fprint(&tmpBuf, fset, &printer.CommentedNode{
		Node:     target,
		Comments: target.Comments,
	})

	// if s := string(tmpBuf.Bytes()); strings.HasSuffix(s, NewLine) {
	// 	if !strings.HasSuffix(s, DoubleNewLine) {
	// 		tmpBuf.WriteString(NewLine)
	// 	}
	// } else {
	// 	tmpBuf.WriteString(DoubleNewLine)
	// }

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

		fmt.Printf("new method will be added: *%s.%s\n", config.Name, funcDecl.Name.Name)
		tmpBuf.WriteString(NewLine)
		tmpBuf.Write(b2[funcDecl.Pos()-1 : funcDecl.End()-1])
		tmpBuf.WriteString(NewLine)
	}
	if tmpBuf.Len() <= 4+len(b1) {
		fmt.Println("no new codes")
		fmt.Println("done")
		return
	}

	f1, err = os.Create("./gen/custom.go")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("writing file")
	tmpBuf.WriteTo(f1)
	f1.Close()
	fmt.Println("done")
}
