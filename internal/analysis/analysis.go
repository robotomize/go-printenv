package analysis

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gituhb.com/robotomize/go-printenv/internal/parser"
	"gituhb.com/robotomize/go-printenv/internal/slice"
	"golang.org/x/mod/modfile"
)

var builtinTypes = []string{
	"string", "int", "int32", "uint32", "int64", "uint64", "bool", "uint16", "int16", "float64", "float32",
	"complex64", "complex128", "byte", "nil", "uint8", "int8", "time.Time", "time.Duration",
}

var _ fmt.Stringer = (*OutputEntry)(nil)

type OutputEntry struct {
	PackageName  string
	EntryName    string
	DefaultValue string
}

func (o OutputEntry) String() string {
	return fmt.Sprintf("%s=%s", o.EntryName, o.DefaultValue)
}

type Option func(*analysis)

type Printer interface {
	Print() []byte
}

type Analyzer interface {
	Analyze() ([]OutputEntry, error)
	Print() []byte
}

type ExtractFunc func(t string) (tagDetail, bool)

func New(pth string, printFunc func(items ...OutputEntry) Printer) Analyzer {
	return &analysis{
		pth: pth, newPrinterFunc: printFunc, extractors: []ExtractFunc{
			extractFunc("env", "prefix", "default"),
		},
		skipDirs: []string{},
	}
}

type node struct {
	refCounter int
	struct1    *parser.Struct
}

type tagDetail struct {
	name         string
	isPrefix     bool
	prefix       string
	defaultValue string
}

type analysis struct {
	pth            string
	modPth         string
	skipDirs       []string
	newPrinterFunc func(items ...OutputEntry) Printer
	extractors     []ExtractFunc
}

func (p *analysis) Print() []byte {
	analyze, _ := p.Analyze()

	return p.newPrinterFunc(analyze...).Print()
}

func (p *analysis) Analyze() ([]OutputEntry, error) {
	var output []OutputEntry
	structs, err := p.parse()
	if err != nil {
		return nil, fmt.Errorf("analyze parse: %w", err)
	}

	structs = slice.Filter(
		structs, func(v *parser.Struct) bool {
			for _, field := range v.Fields {
				var info *tagDetail
				for _, f := range p.extractors {
					detail, ok := f(field.Tag)
					if !ok {
						continue
					}
					info = &detail
					break
				}

				if info == nil {
					return false
				}
			}
			return true
		},
	)

	// if _, err = os.Stat(filepath.Join(dir, "vendor")); err != nil {
	// 	if errors.Is(err, os.ErrNotExist) {
	// 		goPath := os.Getenv("GOPATH")
	// 		pthSegments := filepath.SplitList(goPath)
	// 		pthSegments = append(
	// 			pthSegments, append([]string{"pkg", "mod"}, filepath.SplitList(currGoMod.Mod.Path)...)...,
	// 		)
	// 	}
	//
	// 	return nil, fmt.Errorf("os.Stat: %w", err)
	// }

	// err = walkVendor(
	// 	"", filepath.Join(p.pth, "vendor"), func(modPth, path string) error {
	// 		if strings.HasSuffix(path, ".go") {
	// 			parsed, err := parser.ParseVendorStruct(modPth, path)
	// 			if err != nil {
	// 				return fmt.Errorf("parser.ParseStruct: %w", err)
	// 			}
	//
	// 			structs = append(structs, parsed...)
	// 		}
	//
	// 		return nil
	// 	},
	// )
	// if err != nil {
	// 	return nil, err
	// }

	nodes := make(map[string]*node)
	for _, struct1 := range structs {
		nodes[struct1.PackageName+":"+struct1.Name] = &node{
			refCounter: 0,
			struct1:    struct1,
		}
	}

	for _, struct1 := range structs {
	FieldIter:
		for _, field := range struct1.Fields {
			for _, t := range builtinTypes {
				if field.Type == t {
					continue FieldIter
				}
			}

			// env.Config
			var info *tagDetail
			for _, f := range p.extractors {
				detail, ok := f(field.Tag)
				if !ok {
					continue
				}
				info = &detail
			}

			if info == nil {
				continue
			}

			if strings.Contains(field.Type, ".") {
				continue
			}

			nodPth := struct1.PackageName + ":" + field.Type

			nod, ok := nodes[nodPth]
			if !ok {
				continue
			}

			nod.refCounter += 1
		}
	}

	prepared := slice.Filter(
		slice.MapValuesToSlice(nodes), func(n *node) bool {
			return n.refCounter == 0
		},
	)

	for _, n := range prepared {
		rows, err := p.buildVar("", nodes, n.struct1)
		if err != nil {
			return nil, err
		}

		output = append(output, rows...)
	}

	return output, nil
}

func walkVendor(modPth, dir string, f func(modPth, dir string) error) error {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range dirs {
		if entry.IsDir() {
			if err = walkVendor(
				filepath.Join(modPth, entry.Name()), filepath.Join(dir, entry.Name()), f,
			); err != nil {
				return err
			}
			continue
		}
		if err = f(modPth, filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (p *analysis) buildVar(prefix string, nodes map[string]*node, root *parser.Struct) ([]OutputEntry, error) {
	var output []OutputEntry
FieldIter:
	for _, field := range root.Fields {
		for _, t := range builtinTypes {
			if field.Type == t {
				var info tagDetail
				for _, f := range p.extractors {
					detail, ok := f(field.Tag)
					if !ok {
						continue FieldIter
					}
					info = detail
					break
				}

				output = append(
					output, OutputEntry{
						PackageName:  root.PackageName,
						EntryName:    prefix + info.name,
						DefaultValue: info.defaultValue,
					},
				)
				continue FieldIter
			}
		}

		var info tagDetail
		for _, f := range p.extractors {
			detail, ok := f(field.Tag)
			if !ok {
				continue FieldIter
			}
			info = detail
		}

		if !info.isPrefix {
			output = append(
				output, OutputEntry{
					PackageName:  root.PackageName,
					EntryName:    prefix + info.name,
					DefaultValue: info.defaultValue,
				},
			)
			continue FieldIter
		}

		if !strings.Contains(field.Type, ".") {
			nodPth := root.PackageName + ":" + field.Type
			nod, ok := nodes[nodPth]
			if !ok {
				d, _ := filepath.Split(root.FilePath)
				dirs, err := os.ReadDir(d)
				if err != nil {
					return nil, err
				}

				var structs []*parser.Struct
				for _, entry := range dirs {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
						parsed, err := parser.ParseVendorStruct(filepath.Join(d, entry.Name()))
						if err != nil {
							return nil, fmt.Errorf("parser.ParseVendorStruct: %w", err)
						}

						structs = append(structs, parsed...)
					}
				}

				for _, s := range structs {
					if s.Name == field.Type {
						rows, err := p.buildVar(prefix+info.prefix, nodes, s)
						if err != nil {
							return nil, err
						}

						output = append(output, rows...)

						continue FieldIter
					}
				}

				continue
			}

			rows, err := p.buildVar(prefix+info.prefix, nodes, nod.struct1)
			if err != nil {
				return nil, err
			}
			output = append(output, rows...)
			continue
		}

		depStruct, exist, err := p.parseDepStructPth(nodes, field, root)
		if err != nil {
			return nil, err
		}

		if !exist {
			continue
		}

		rows, err := p.buildVar(prefix+info.prefix, nodes, depStruct)
		if err != nil {
			return nil, err
		}
		output = append(output, rows...)
	}

	return output, nil
}

func extractFunc(key, prefixFlag, defaultFlag string) func(t string) (tagDetail, bool) {
	return func(t string) (tagDetail, bool) {
		var detail tagDetail
		name, flags, ok := parser.LookupTagValue(t, key)
		if !ok {
			return detail, ok
		}

		detail.name = name

		for _, flag := range flags {
			switch {
			case strings.Contains(flag, prefixFlag):
				detail.isPrefix = true
				tokens := strings.Split(flag, "=")
				if len(tokens) > 1 {
					detail.prefix = tokens[1]
				}
			case strings.Contains(flag, defaultFlag):
				tokens := strings.Split(flag, "=")
				if len(tokens) > 1 {
					detail.defaultValue = tokens[1]
				}
			default:
			}
		}

		return detail, true
	}
}

var ErrGoPkgNotFound = errors.New("GOPATH not found")

func (p *analysis) parseDepStructPth(
	nodes map[string]*node,
	field *parser.StructField,
	struct1 *parser.Struct,
) (*parser.Struct, bool, error) {
	pth, err := CurrentVendorRootPth(p.pth)
	if err != nil {
		return nil, false, fmt.Errorf("CurrentVendorRootPth: %w", err)
	}

	tokens := strings.Split(field.Type, ".")
	if len(tokens) > 1 {
		pkgName := tokens[0]
		structTyp := tokens[1]

		for _, pkg := range struct1.Imports {
			if strings.Contains(pkg, pkgName) {
				nodPth := pkg + ":" + structTyp
				nod, ok := nodes[nodPth]
				if ok {
					return nod.struct1, true, nil
				}

				var structs []*parser.Struct

				vendorPth := filepath.Join(append([]string{pth}, filepath.SplitList(pkg)...)...)
				if _, err = os.Stat(filepath.Join(append([]string{pth}, filepath.SplitList(pkg)...)...)); err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						return nil, false, err
					}

					pkgSegments := filepath.SplitList(pkg)
					vendorPth = filepath.Join(append([]string{pth}, pkgSegments[:len(pkgSegments)-1]...)...)
				}

				dirs, err := os.ReadDir(vendorPth)
				if err != nil {
					return nil, false, err
				}

				for _, d := range dirs {
					if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
						parsed, err := parser.ParseVendorStruct(filepath.Join(vendorPth, d.Name()))
						if err != nil {
							return nil, false, fmt.Errorf("parser.ParseVendorStruct: %w", err)
						}

						structs = append(structs, parsed...)
					}
				}

				for _, s := range structs {
					if s.Name == structTyp {
						return s, true, nil
					}
				}

				return nil, false, nil
			}
		}
	}

	return nil, false, nil
}

func (p *analysis) parse() ([]*parser.Struct, error) {
	var structs []*parser.Struct

	modPth := filepath.Join(p.pth, "go.mod")
	src, err := os.ReadFile(modPth)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	parse, err := modfile.Parse(modPth, src, nil)
	if err != nil {
		return nil, fmt.Errorf("modfile.Parse: %w", err)
	}

	if parse.Module == nil {
		return nil, ErrGoModNotFound
	}

	p.modPth = parse.Module.Mod.Path

	if err = p.walkDir(
		p.pth, func(modPth, rootPth, path string) error {
			if strings.HasSuffix(path, ".go") {
				parsed, err := parser.ParseStruct(modPth, rootPth, path)
				if err != nil {
					return fmt.Errorf("parser.ParseStruct: %w", err)
				}

				structs = append(structs, parsed...)
			}

			return nil
		},
	); err != nil {
		return nil, fmt.Errorf("walkDir: %w", err)
	}

	return structs, nil
}

var ErrGoModNotFound = errors.New("go mod not found")

func (p *analysis) walkDir(dir string, f func(modPth, rootPth, pth string) error) error {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("os.ReadDir: %w", err)
	}

	for _, entry := range dirs {
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "..") {
			continue
		}

		if entry.IsDir() && entry.Name() != "vendor" {
			if err = p.walkDir(filepath.Join(dir, entry.Name()), f); err != nil {
				return err
			}
			continue
		}

		if err = f(p.modPth, p.pth, filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}
