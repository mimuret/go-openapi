// +build ignore

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"reflect"
)

func main() {
	var buf bytes.Buffer
	outf := func(format string, args ...interface{}) {
		fmt.Fprintf(&buf, format, args...)
	}

	outf("// Code generated by mksetroot.go. DO NOT EDIT.")
	outf("\n\npackage openapi")

	f, err := parser.ParseFile(token.NewFileSet(), "interfaces.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if genDecl.Doc == nil || len(genDecl.Doc.List) == 0 || genDecl.Doc.List[0].Text != "//+object" {
			log.Printf("%v is not an openapi object. skip.", genDecl.Specs[0].(*ast.TypeSpec).Name.Name)
			continue
		}

		for _, spec := range genDecl.Specs {
			typ, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			st, ok := typ.Type.(*ast.StructType)
			if !ok {
				continue
			}

			log.Printf("generate %s.setRoot()", typ.Name.Name)
			outf("\n\nfunc (v *%s) setRoot(root *OpenAPI) {", typ.Name.Name)

			for _, field := range st.Fields.List {
				switch t := field.Type.(type) {
				case *ast.Ident, *ast.InterfaceType:
					// nothing to do
				case *ast.StarExpr: // pointer to struct
					if field.Names[0].Name == "root" {
						outf("\nv.root = root")
						continue
					}
					outf("\nif v.%s != nil {", field.Names[0].Name)
					outf("\nv.%s.setRoot(root)", field.Names[0].Name)
					outf("\n}")
				case *ast.ArrayType:
					switch t.Elt.(type) {
					case *ast.Ident, *ast.InterfaceType:
						// nothing to do
					case *ast.StarExpr:
						outf("\nfor i := range v.%s {", field.Names[0].Name)
						outf("\nv.%s[i].setRoot(root)", field.Names[0].Name)
						outf("\n}")
					default:
						log.Print(reflect.TypeOf(t.Elt))
					}
				case *ast.MapType:
					switch tv := t.Value.(type) {
					case *ast.Ident, *ast.InterfaceType:
						// nothing to do
					case *ast.StarExpr:
						outf("\nfor k := range v.%s {", field.Names[0].Name)
						outf("\nv.%s[k].setRoot(root)", field.Names[0].Name)
						outf("\n}")
					case *ast.ArrayType:
						switch tv.Elt.(type) {
						case *ast.StarExpr:
							outf("\nfor k := range v.%s {", field.Names[0].Name)
							outf("\nfor i := range v.%s[k] {", field.Names[0].Name)
							outf("\nv.%s[k][i].setRoot(root)", field.Names[0].Name)
							outf("\n}")
							outf("\n}")
						}
					default:
						log.Print(reflect.TypeOf(t.Value))
					}
				default:
					log.Printf("%s %s", field.Type, reflect.TypeOf(field.Type))
				}
			}

			outf("\n}")
		}
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Printf("%s\n", buf.Bytes())

		log.Fatalf("error on formatting: %+v", err)
	}
	if err := ioutil.WriteFile("setroot_gen.go", src, 0644); err != nil {
		log.Fatal(err)
	}
}
