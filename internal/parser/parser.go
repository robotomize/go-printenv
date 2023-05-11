package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type File struct {
	FilePath    string
	PackageName string
	Imports     []string
	Fields      []Field
}

type Field struct {
	StructName  string
	FieldName   string
	FieldType   string
	CommentName string
	TagValue    string
}

type Struct struct {
	FilePath    string
	PackageName string
	Imports     []string
	Name        string
	Fields      []*StructField
}

type StructField struct {
	Name string
	Type string
	Tag  string
}

func ParseVendorStruct(pth string) ([]*Struct, error) {
	file, err := ParseVendor(pth)
	if err != nil {
		return nil, fmt.Errorf("ParseVendor: %w", err)
	}

	var structs []*Struct
FieldIter:
	for _, field := range file.Fields {
		for _, struct1 := range structs {
			if field.StructName == struct1.Name {
				struct1.Fields = append(
					struct1.Fields, &StructField{
						Name: field.FieldName,
						Type: field.FieldType,
						Tag:  field.TagValue,
					},
				)
				continue FieldIter
			}
		}
		structs = append(
			structs, &Struct{
				FilePath:    file.FilePath,
				PackageName: file.PackageName,
				Imports:     file.Imports,
				Name:        field.StructName,
				Fields: []*StructField{
					{
						Name: field.FieldName,
						Type: field.FieldType,
						Tag:  field.TagValue,
					},
				},
			},
		)
	}

	return structs, nil
}

func ParseStruct(modPth, rootPth, pth string) ([]*Struct, error) {
	file, err := Parse1(modPth, rootPth, pth)
	if err != nil {
		return nil, err
	}
	var structs []*Struct
FieldIter:
	for _, field := range file.Fields {
		for _, struct1 := range structs {
			if field.StructName == struct1.Name {
				struct1.Fields = append(
					struct1.Fields, &StructField{
						Name: field.FieldName,
						Type: field.FieldType,
						Tag:  field.TagValue,
					},
				)
				continue FieldIter
			}
		}
		structs = append(
			structs, &Struct{
				FilePath:    file.FilePath,
				PackageName: file.PackageName,
				Imports:     file.Imports,
				Name:        field.StructName,
				Fields: []*StructField{
					{
						Name: field.FieldName,
						Type: field.FieldType,
						Tag:  field.TagValue,
					},
				},
			},
		)
	}

	return structs, nil
}

// /root/vendor

func ParseVendor(source string) (*File, error) {
	fSet := token.NewFileSet()
	src, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	file, err := parser.ParseFile(fSet, source, src, 0)
	if err != nil {
		return nil, err
	}

	packageName := file.Name.Name

	pFile := &File{
		FilePath:    source,
		PackageName: packageName,
		Fields:      make([]Field, 0),
	}

	ast.Inspect(
		file, func(n ast.Node) bool {
			switch t := n.(type) {
			case *ast.TypeSpec:
				e, ok := t.Type.(*ast.StructType)
				if ok && token.IsExported(t.Name.Name) {
					if e.Fields == nil || e.Fields.NumFields() < 1 {
						// skip empty structs
						return true
					}

					for _, field := range e.Fields.List {
						if len(field.Names) == 0 || !token.IsExported(field.Names[0].Name) || field.Tag == nil {
							continue
						}

						pFile.Fields = append(
							pFile.Fields, Field{
								StructName:  t.Name.Name,
								FieldName:   field.Names[0].Name,
								CommentName: field.Names[0].Name,
								FieldType:   string(src[field.Type.Pos()-1 : field.Type.End()-1]),
								TagValue:    field.Tag.Value,
							},
						)
					}
				}
			case *ast.ImportSpec:
				pFile.Imports = append(pFile.Imports, strings.Trim(t.Path.Value, "\""))
			}

			return true
		},
	)

	return pFile, nil
}

func Parse1(modPth, rootPth, source string) (*File, error) {
	dir, _ := filepath.Split(source)

	pkg := filepath.Clean(strings.Replace(dir, rootPth, modPth, -1))

	fSet := token.NewFileSet()
	src, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	file, err := parser.ParseFile(fSet, source, src, 0)
	if err != nil {
		return nil, err
	}

	packageName := file.Name.Name
	if packageName == "main" {
		packageName = modPth
	} else {
		packageName = pkg
	}

	// tracing.Config
	pFile := &File{
		FilePath:    source,
		PackageName: packageName,
		Fields:      make([]Field, 0),
	}

	ast.Inspect(
		file, func(n ast.Node) bool {
			switch t := n.(type) {
			case *ast.TypeSpec:
				e, ok := t.Type.(*ast.StructType)
				if ok && token.IsExported(t.Name.Name) {
					if e.Fields == nil || e.Fields.NumFields() < 1 {
						// skip empty structs
						return true
					}

					for _, field := range e.Fields.List {
						if len(field.Names) == 0 || !token.IsExported(field.Names[0].Name) || field.Tag == nil {
							continue
						}

						pFile.Fields = append(
							pFile.Fields, Field{
								StructName:  t.Name.Name,
								FieldName:   field.Names[0].Name,
								CommentName: field.Names[0].Name,
								FieldType:   string(src[field.Type.Pos()-1 : field.Type.End()-1]),
								TagValue:    field.Tag.Value,
							},
						)
					}
				}
			case *ast.ImportSpec:
				pFile.Imports = append(pFile.Imports, strings.Trim(t.Path.Value, "\""))
			}

			return true
		},
	)

	return pFile, nil
}

func LookupTagValue(tag, key string) (name string, flags []string, ok bool) {
	raw := strings.Trim(tag, "`")

	value, ok := reflect.StructTag(raw).Lookup(key)
	if !ok {
		return value, nil, ok
	}

	values := strings.Split(value, ",")

	if len(values) < 1 {
		return "", nil, true
	}

	return values[0], values[1:], true
}

func HasTagFlag(flags []string, query string) bool {
	for _, flag := range flags {
		if flag == query {
			return true
		}
	}

	return false
}
