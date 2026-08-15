// Command mockscan lists Go source files that declare top-level interfaces.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

type pathList []string

func (paths *pathList) String() string {
	return strings.Join(*paths, ",")
}

func (paths *pathList) Set(value string) error {
	*paths = append(*paths, value)
	return nil
}

func main() {
	root := flag.String("root", "", "project root to scan")
	var excludes pathList
	flag.Var(&excludes, "exclude", "directory tree to exclude")
	flag.Parse()

	if *root == "" {
		log.Fatal("-root is required")
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		log.Fatalf("resolve root: %v", err)
	}
	for index, path := range excludes {
		excludePath, err := filepath.Abs(path)
		if err != nil {
			log.Fatalf("resolve excluded path %q: %v", path, err)
		}
		excludes[index] = excludePath
	}

	if err := scan(rootPath, excludes); err != nil {
		log.Fatal(err)
	}
}

// scan emits slash-separated project-relative paths for source files with interfaces.
func scan(root string, excludes []string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDirectory(path, entry.Name(), excludes) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		hasInterface, err := declaresExportedInterface(path)
		if err != nil {
			return err
		}
		if !hasInterface {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("make source path relative: %w", err)
		}
		fmt.Println(filepath.ToSlash(relativePath))
		return nil
	})
}

// shouldSkipDirectory keeps generated, vendored, and explicitly excluded code out of scans.
func shouldSkipDirectory(path, name string, excludes []string) bool {
	if name == ".git" || name == "vendor" || name == ".mocks" {
		return true
	}
	for _, exclude := range excludes {
		if isWithin(path, exclude) {
			return true
		}
	}
	return false
}

// isWithin reports whether path equals or belongs to parent.
func isWithin(path, parent string) bool {
	relativePath, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return relativePath == "." || (relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator)))
}

// declaresExportedInterface reports whether a Go file declares an exported top-level interface type.
func declaresExportedInterface(path string) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false, fmt.Errorf("parse %q: %w", path, err)
	}
	for _, declaration := range file.Decls {
		generalDeclaration, ok := declaration.(*ast.GenDecl)
		if !ok || generalDeclaration.Tok != token.TYPE {
			continue
		}
		for _, specification := range generalDeclaration.Specs {
			typeSpecification, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if ast.IsExported(typeSpecification.Name.Name) {
				if _, ok := typeSpecification.Type.(*ast.InterfaceType); ok {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
