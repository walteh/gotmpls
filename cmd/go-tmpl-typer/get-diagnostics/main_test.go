package get_diagnostics

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/go-tmpl-typer/pkg/ast"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/parser"
	"github.com/walteh/go-tmpl-typer/pkg/types"
	"gitlab.com/tozd/go/errors"
)

type Person struct {
	Name string
	Age  int
}

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) GetAge() int {
	return p.Age
}

func (p *Person) IsAdult() bool {
	return p.Age >= 18
}

func setupTestTemplate(t *testing.T) (string, func()) {
	t.Helper()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "tmpl-test-*")
	require.NoError(t, err)

	// Create types directory and package
	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	// Create go.mod file
	goModContent := []byte(`module test

go 1.21
`)
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), goModContent, 0644)
	require.NoError(t, err)

	// Create person.go file
	personGoContent := []byte(`package types

type Person struct {
	Name string
	Age  int
}

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) GetAge() int {
	return p.Age
}

func (p *Person) IsAdult() bool {
	return p.Age >= 18
}
`)
	err = os.WriteFile(filepath.Join(typesDir, "person.go"), personGoContent, 0644)
	require.NoError(t, err)

	// Create template file
	templateContent := []byte(`{{- /*gotype: test/types.Person */ -}}

Name: {{.Name}}
Age: {{.Age}}

{{if .IsAdult}}
Adult Status: {{.GetName | printf "%s is an adult" | upper}}
{{else}}
Adult Status: {{.GetName | printf "%s is not an adult" | upper}}
{{end}}
`)
	templatePath := filepath.Join(tmpDir, "person.tmpl")
	err = os.WriteFile(templatePath, templateContent, 0644)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return templatePath, cleanup
}

func getDiagnostics(ctx context.Context, templatePath string) (*diagnostic.Diagnostics, error) {
	// Read the template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, errors.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Create components
	templateParser := parser.NewDefaultTemplateParser()
	typeValidator := types.NewDefaultValidator()
	packageAnalyzer := ast.NewDefaultPackageAnalyzer()

	// Get the package directory (parent of the template file)
	packageDir := filepath.Dir(templatePath)

	// Analyze the package
	registry, err := packageAnalyzer.AnalyzePackage(ctx, packageDir)
	if err != nil {
		return nil, errors.Errorf("failed to analyze package: %w", err)
	}

	// Parse the template
	info, err := templateParser.Parse(ctx, content, templatePath)
	if err != nil {
		return &diagnostic.Diagnostics{
			Errors: []diagnostic.Diagnostic{
				{
					Message:  "template parse error: " + err.Error(),
					Line:     1,
					Column:   1,
					EndLine:  1,
					EndCol:   1,
					Severity: diagnostic.Error,
				},
			},
		}, nil
	}

	// Generate diagnostics
	generator := diagnostic.NewDefaultGenerator()
	return generator.Generate(ctx, info, typeValidator, registry)
}

func TestGetDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     *diagnostic.Diagnostics
		wantErr  bool
	}{
		{
			name:     "person template",
			template: "person.tmpl",
			want: &diagnostic.Diagnostics{
				Errors:   []diagnostic.Diagnostic{},
				Warnings: []diagnostic.Diagnostic{},
				Hints: []diagnostic.Diagnostic{
					{
						Message:  "Type: string",
						Line:     8,
						Column:   9,
						EndLine:  8,
						EndCol:   13,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Type: int",
						Line:     9,
						Column:   8,
						EndLine:  9,
						EndCol:   11,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Type: string",
						Line:     11,
						Column:   21,
						EndLine:  12,
						EndCol:   5,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Type: string",
						Line:     12,
						Column:   19,
						EndLine:  14,
						EndCol:   4,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Type: func() bool",
						Line:     14,
						Column:   6,
						EndLine:  14,
						EndCol:   12,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Type: func() string",
						Line:     15,
						Column:   8,
						EndLine:  15,
						EndCol:   14,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     15,
						Column:   17,
						EndLine:  15,
						EndCol:   22,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     16,
						Column:   9,
						EndLine:  16,
						EndCol:   15,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     16,
						Column:   30,
						EndLine:  16,
						EndCol:   35,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     17,
						Column:   9,
						EndLine:  17,
						EndCol:   15,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     17,
						Column:   22,
						EndLine:  17,
						EndCol:   27,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     17,
						Column:   30,
						EndLine:  17,
						EndCol:   36,
						Severity: diagnostic.Hint,
					},
					{
						Message:  "Returns: string",
						Line:     17,
						Column:   44,
						EndLine:  17,
						EndCol:   49,
						Severity: diagnostic.Hint,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			templatePath := filepath.Join("..", "..", "..", "examples", tt.template)

			got, err := getDiagnostics(ctx, templatePath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDiagnosticsWithTempFile(t *testing.T) {
	templatePath, cleanup := setupTestTemplate(t)
	defer cleanup()

	// Test diagnostics
	ctx := context.Background()
	got, err := getDiagnostics(ctx, templatePath)
	require.NoError(t, err)

	want := &diagnostic.Diagnostics{
		Errors:   []diagnostic.Diagnostic{},
		Warnings: []diagnostic.Diagnostic{},
		Hints: []diagnostic.Diagnostic{
			{
				Message:  "Type: string",
				Line:     3,
				Column:   9,
				EndLine:  3,
				EndCol:   13,
				Severity: diagnostic.Hint,
			},
			{
				Message:  "Type: int",
				Line:     4,
				Column:   8,
				EndLine:  4,
				EndCol:   11,
				Severity: diagnostic.Hint,
			},
			{
				Message:  "Type: func() bool",
				Line:     6,
				Column:   6,
				EndLine:  6,
				EndCol:   13,
				Severity: diagnostic.Hint,
			},
			{
				Message:  "Type: func() string",
				Line:     7,
				Column:   17,
				EndLine:  7,
				EndCol:   24,
				Severity: diagnostic.Hint,
			},
			{
				Message:  "Returns: string",
				Line:     7,
				Column:   27,
				EndLine:  7,
				EndCol:   33,
				Severity: diagnostic.Hint,
			},
			{
				Message:  "Returns: string",
				Line:     7,
				Column:   53,
				EndLine:  7,
				EndCol:   58,
				Severity: diagnostic.Hint,
			},
			{
				Message:  "Returns: string",
				Line:     9,
				Column:   27,
				EndLine:  9,
				EndCol:   33,
				Severity: diagnostic.Hint,
			},
			{
				Message:  "Returns: string",
				Line:     9,
				Column:   57,
				EndLine:  9,
				EndCol:   62,
				Severity: diagnostic.Hint,
			},
		},
	}

	assert.Equal(t, want, got)

	// Test template execution
	content, err := os.ReadFile(templatePath)
	require.NoError(t, err)

	// Create template with necessary functions
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"upper":  strings.ToUpper,
		"printf": fmt.Sprintf,
	}).Parse(string(content))
	require.NoError(t, err)

	// Test adult case
	adultPerson := &Person{
		Name: "John",
		Age:  25,
	}
	var adultBuf bytes.Buffer
	err = tmpl.Execute(&adultBuf, adultPerson)
	require.NoError(t, err)

	wantAdult := `Name: John
Age: 25


Adult Status: JOHN IS AN ADULT

`
	assert.Equal(t, wantAdult, adultBuf.String())

	// Test non-adult case
	childPerson := &Person{
		Name: "Jane",
		Age:  15,
	}
	var childBuf bytes.Buffer
	err = tmpl.Execute(&childBuf, childPerson)
	require.NoError(t, err)

	wantChild := `Name: Jane
Age: 15


Adult Status: JANE IS NOT AN ADULT

`
	assert.Equal(t, wantChild, childBuf.String())
}
