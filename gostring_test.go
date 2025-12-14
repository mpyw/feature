package feature_test

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mpyw/feature"
)

// TestGoString tests the GoString method for keys.
func TestGoString(t *testing.T) {
	t.Parallel()

	t.Run("Key GoString returns valid Go expression", func(t *testing.T) {
		t.Parallel()

		key := feature.NewNamed[string]("test-key")
		goStr := key.GoString()

		want := `feature.New[string](feature.WithName("test-key"))`
		if goStr != want {
			t.Errorf("GoString() = %q, want %q", goStr, want)
		}

		assertCompilesWithFeatureImport(t, goStr)
	})

	t.Run("Key GoString with int type", func(t *testing.T) {
		t.Parallel()

		key := feature.NewNamed[int]("max-retries")
		goStr := key.GoString()

		want := `feature.New[int](feature.WithName("max-retries"))`
		if goStr != want {
			t.Errorf("GoString() = %q, want %q", goStr, want)
		}

		assertCompilesWithFeatureImport(t, goStr)
	})

	t.Run("BoolKey GoString returns valid Go expression", func(t *testing.T) {
		t.Parallel()

		flag := feature.NewNamedBool("my-feature")
		goStr := flag.GoString()

		want := `feature.NewBool(feature.WithName("my-feature"))`
		if goStr != want {
			t.Errorf("GoString() = %q, want %q", goStr, want)
		}

		assertCompilesWithFeatureImport(t, goStr)
	})
}

// assertCompilesWithFeatureImport verifies that the given expression compiles
// as valid Go code with the feature package imported.
func assertCompilesWithFeatureImport(t *testing.T, expr string) {
	t.Helper()

	// Wrap the expression in a minimal Go source file
	src := `package main

import "github.com/mpyw/feature"

var _ = ` + expr + `
`

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Errorf("GoString() output %q failed to parse: %v", expr, err)

		return
	}

	// Type-check the parsed file
	conf := types.Config{
		Importer:                 newFeatureImporter(t, fset),
		GoVersion:                "",
		Context:                  nil,
		IgnoreFuncBodies:         false,
		FakeImportC:              false,
		Error:                    nil,
		Sizes:                    nil,
		DisableUnusedImportCheck: false,
	}

	_, err = conf.Check("main", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Errorf("GoString() output %q failed type-check: %v", expr, err)
	}
}

// featureImporter implements types.Importer with support for the feature package.
type featureImporter struct {
	t           *testing.T
	fset        *token.FileSet
	defaultImpl types.Importer
	featurePkg  *types.Package
	projectRoot string
}

func newFeatureImporter(t *testing.T, fset *token.FileSet) *featureImporter {
	t.Helper()

	return &featureImporter{
		t:           t,
		fset:        fset,
		defaultImpl: importer.Default(),
		featurePkg:  nil,
		projectRoot: findProjectRoot(t),
	}
}

func (fi *featureImporter) Import(path string) (*types.Package, error) {
	if path == "github.com/mpyw/feature" {
		if fi.featurePkg != nil {
			return fi.featurePkg, nil
		}

		pkg, err := fi.loadFeaturePackage()
		if err != nil {
			return nil, err
		}

		fi.featurePkg = pkg

		return pkg, nil
	}

	// Standard library
	pkg, err := fi.defaultImpl.Import(path)
	if err != nil {
		panic("unexpected import: " + path)
	}

	return pkg, nil
}

func (fi *featureImporter) loadFeaturePackage() (*types.Package, error) {
	entries, err := os.ReadDir(fi.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("reading project root: %w", err)
	}

	// Count .go files (excluding tests) for pre-allocation
	var count int

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			count++
		}
	}

	files := make([]*ast.File, 0, count)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := fi.parseFile(name)
		if err != nil {
			return nil, err
		}

		files = append(files, file)
	}

	// Type-check the feature package
	conf := types.Config{
		Importer:                 fi.defaultImpl,
		GoVersion:                "",
		Context:                  nil,
		IgnoreFuncBodies:         false,
		FakeImportC:              false,
		Error:                    nil,
		Sizes:                    nil,
		DisableUnusedImportCheck: false,
	}

	pkg, err := conf.Check("github.com/mpyw/feature", fi.fset, files, nil)
	if err != nil {
		return nil, fmt.Errorf("type-checking feature package: %w", err)
	}

	return pkg, nil
}

func (fi *featureImporter) parseFile(name string) (*ast.File, error) {
	path := filepath.Join(fi.projectRoot, name)

	src, err := os.ReadFile(path) //#nosec G304 -- path is constructed from project root
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}

	file, err := parser.ParseFile(fi.fset, name, src, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}

	return file, nil
}

// WARNING: GoString outputs a valid Go expression that creates an equivalent key,
// but the resulting key will have a different identity (pointer address).
// Two keys created from the same GoString output will NOT be equal.
func ExampleKey_GoString() {
	key := feature.NewNamed[string]("api-key")
	fmt.Println(key.GoString())

	// Output:
	// feature.New[string](feature.WithName("api-key"))
}

// WARNING: GoString outputs a valid Go expression that creates an equivalent key,
// but the resulting key will have a different identity (pointer address).
// Two keys created from the same GoString output will NOT be equal.
func ExampleBoolKey_GoString() {
	flag := feature.NewNamedBool("debug-mode")
	fmt.Println(flag.GoString())

	// Output:
	// feature.NewBool(feature.WithName("debug-mode"))
}

// findProjectRoot returns the absolute path to the project root directory.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find project root (go.mod)")
		}

		dir = parent
	}
}
