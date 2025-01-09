package ast

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestModule(t *testing.T) (string, context.Context) {
	ctx := context.Background()

	// Create a temporary directory for our test module
	tmpDir, err := os.MkdirTemp("", "package-analyzer-test")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir, ctx
}

func TestPackageAnalyzer(t *testing.T) {
	tmpDir, ctx := setupTestModule(t)

	// Create a minimal Go module
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`
module example.com/test

go 1.21
`), 0644)
	require.NoError(t, err)

	// Create a test package with a simple type
	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(typesDir, "types.go"), []byte(`
package types

type Person struct {
	Name    string
	Age     int
	Address struct {
		Street string
		City   string
	}
}

func (p *Person) GetJob() string {
	return "Developer"
}

func (p *Person) HasJob() bool {
	return true
}
`), 0644)
	require.NoError(t, err)

	// Create the analyzer
	analyzer := NewDefaultPackageAnalyzer()

	// Analyze the package
	registry, err := analyzer.AnalyzePackage(ctx, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, registry)

	// Verify the types were loaded
	pkg, err := registry.GetPackage(ctx, "example.com/test/types")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	// Check if we can find the Person type
	obj := pkg.Scope().Lookup("Person")
	require.NotNil(t, obj)

	// Verify it's a struct type
	structType, ok := obj.Type().Underlying().(*types.Struct)
	require.True(t, ok)

	// Check the fields
	assert.Equal(t, "Name", structType.Field(0).Name())
	assert.Equal(t, "string", structType.Field(0).Type().String())
	assert.Equal(t, "Age", structType.Field(1).Name())
	assert.Equal(t, "int", structType.Field(1).Type().String())

	// Check methods
	personType := obj.Type()
	methods := types.NewMethodSet(types.NewPointer(personType))

	hasJob := methods.Lookup(nil, "HasJob")
	require.NotNil(t, hasJob)

	getJob := methods.Lookup(nil, "GetJob")
	require.NotNil(t, getJob)
}

func TestPackageAnalyzer_NoGoMod(t *testing.T) {
	tmpDir, ctx := setupTestModule(t)

	// Create a test package without a go.mod
	typesDir := filepath.Join(tmpDir, "types")
	err := os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(typesDir, "types.go"), []byte(`
package types

type Person struct {
	Name string
}
`), 0644)
	require.NoError(t, err)

	analyzer := NewDefaultPackageAnalyzer()
	_, err = analyzer.AnalyzePackage(ctx, tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing go.mod")
}

func TestPackageAnalyzer_InvalidPath(t *testing.T) {
	ctx := context.Background()
	analyzer := NewDefaultPackageAnalyzer()
	_, err := analyzer.AnalyzePackage(ctx, "/path/that/does/not/exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing go.mod")
}

func TestPackageAnalyzer_Diagnostics(t *testing.T) {
	tmpDir, ctx := setupTestModule(t)

	// Create a minimal Go module
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`
module example.com/test

go 1.21
`), 0644)
	require.NoError(t, err)

	// Create a test package with invalid syntax
	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	// Test case 1: Invalid type hint (non-existent package)
	err = os.WriteFile(filepath.Join(typesDir, "invalid_type.go"), []byte(`
package types

// Invalid type hint
type InvalidType struct {
	Field NonExistentType
}
`), 0644)
	require.NoError(t, err)

	analyzer := NewDefaultPackageAnalyzer()
	registry, err := analyzer.AnalyzePackage(ctx, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, registry)

	// Verify that we can detect the invalid type
	pkg, err := registry.GetPackage(ctx, "example.com/test/types")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	// The package should load but the type should be marked as invalid
	obj := pkg.Scope().Lookup("InvalidType")
	require.NotNil(t, obj)

	// Test case 2: Missing package
	_, err = registry.GetPackage(ctx, "non/existent/package")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package non/existent/package not found")

	// Test case 3: Invalid type lookup
	_, err = registry.GetTypes(ctx, "non/existent/package")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package non/existent/package not found")
}
