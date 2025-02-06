package generator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
	"gitlab.com/tozd/go/errors"
)

// File represents a generated file
type File struct {
	Path     string // Relative path to write the file
	Contents string // File contents
}

// Generator generates Go code from a VSCode meta model.
type Generator struct {
	model      vscodemetamodel.MetaModel
	outputPath string
}

// NewGenerator creates a new LSP protocol code generator
func NewGenerator(model vscodemetamodel.MetaModel, outputPath string) *Generator {
	return &Generator{
		model:      model,
		outputPath: outputPath,
	}
}

// GenerateFiles generates all Go files for the types
func (g *Generator) GenerateFiles(ctx context.Context, packageName string) ([]File, error) {
	var files []File

	// Generate types file
	typesFile, err := g.generateTypesFile(packageName)
	if err != nil {
		return nil, errors.Errorf("generating types file: %w", err)
	}
	files = append(files, typesFile)

	return files, nil
}

// generateTypesFile generates the main types file
func (g *Generator) generateTypesFile(packageName string) (File, error) {
	var buf bytes.Buffer

	// File header
	header := fmt.Sprintf(`// Code generated by lsproto-revamp. DO NOT EDIT.

package %s

import (
	"bytes"
	"encoding/json"

	"gitlab.com/tozd/go/errors"
)

`, packageName)
	buf.WriteString(header)

	// Process each request
	for i := range g.model.Requests {
		// Get type info for the request
		info, err := g.getTypeInfo(&g.model.Requests[i])
		if err != nil {
			return File{}, errors.Errorf("getting type info for request: %w", err)
		}

		// Generate request type
		if err := g.generateType(&buf, &info); err != nil {
			return File{}, errors.Errorf("generating request type: %w", err)
		}

		// Generate result type if present
		if info.Result != nil {
			// First generate the union type (Ors)
			resultInfo := info.Result
			// Remove "Request" from the type name for result types
			baseName := strings.TrimSuffix(info.Name, "Request")
			unionTypeName := fmt.Sprintf("%sResultOrs", baseName)
			wrapperTypeName := fmt.Sprintf("%sResult", baseName)

			// Set the union type name
			resultInfo.Name = unionTypeName

			// Generate union type
			if err := g.generateType(&buf, resultInfo); err != nil {
				return File{}, errors.Errorf("generating result union type: %w", err)
			}

			// Then generate the wrapper type that embeds it
			wrapperInfo := &TypeInfo{
				Name:         wrapperTypeName,
				EmbeddedType: unionTypeName,
			}

			// Generate wrapper type
			if err := g.generateType(&buf, wrapperInfo); err != nil {
				return File{}, errors.Errorf("generating result wrapper type: %w", err)
			}
		}
	}

	return File{
		Path:     filepath.Join("types.go"),
		Contents: buf.String(),
	}, nil
}

// Template function map
var templateFuncs = template.FuncMap{
	"lower": strings.ToLower,
}

// generateType generates code for a single type
func (g *Generator) generateType(buf *bytes.Buffer, info *TypeInfo) error {
	// Add documentation
	if info.Documentation != "" {
		buf.WriteString(fmt.Sprintf("// %s\n", info.Documentation))
	}

	if info.IsRequest || (info.EmbeddedType != "" && !info.IsUnion) {
		// Generate type with embedding
		tmpl := template.Must(template.New("request").Parse(`
type {{.Name}}Request struct {
	{{.EmbeddedType}}
}

type {{.Name}}ResultOrs struct {
	{{- range .UnionFields}}
	{{.Name}} {{if .IsPointer}}*{{end}}{{if .IsArray}}[]{{end}}{{.Type}}
	{{- end}}
	IsNull bool
}

func (r {{.Name}}ResultOrs) MarshalJSON() ([]byte, error) {
	if r.IsNull {
		return json.Marshal(nil)
	}

	{{- range .UnionFields}}
	if r.{{.Name}} != nil {
		return json.Marshal(*r.{{.Name}})
	}
	{{- end}}

	return nil, errors.New("invalid {{.Name}}ResultOrs")
}

func (r *{{.Name}}ResultOrs) UnmarshalJSON(data []byte) error {
	// Try null
	if bytes.Equal(data, []byte("null")) {
		r.IsNull = true
		return nil
	}

	{{- range .UnionFields}}
	// Try {{.Type}}
	var {{.Name | lower}} {{if .IsArray}}[]{{end}}{{.Type}}
	if err := json.Unmarshal(data, &{{.Name | lower}}); err == nil {
		r.{{.Name}} = &{{.Name | lower}}
		return nil
	}
	{{- end}}

	return errors.New("invalid {{.Name}}ResultOrs")
}

type {{.Name}}Result struct {
	{{.Name}}ResultOrs
}
`))

		if err := tmpl.Execute(buf, info); err != nil {
			return errors.Errorf("executing union template: %w", err)
		}
	}

	return nil
}

func (g *Generator) Generate() error {
	buf := &bytes.Buffer{}

	for i := range g.model.Requests {
		info, err := g.getTypeInfo(&g.model.Requests[i])
		if err != nil {
			return errors.Errorf("getting request type info: %w", err)
		}

		// Generate type with embedding
		tmpl := template.Must(template.New("request").Parse(`
type {{.Name}}Request struct {
	{{.EmbeddedType}}
}
`))
		if err := tmpl.Execute(buf, info); err != nil {
			return errors.Errorf("executing request template: %w", err)
		}

		// Generate result type
		resultInfo := info.Result

		// Generate union type
		tmpl = template.Must(template.New("union").Funcs(templateFuncs).Parse(`
type {{.Name}}ResultOrs struct {
	{{- range .UnionFields}}
	{{.Name}} {{if .IsPointer}}*{{end}}{{if .IsArray}}[]{{end}}{{.Type}}
	{{- end}}
	IsNull bool
}

func (r {{.Name}}ResultOrs) MarshalJSON() ([]byte, error) {
	if r.IsNull {
		return json.Marshal(nil)
	}
	{{- range .UnionFields}}
	if r.{{.Name}} != nil {
		return json.Marshal({{if .IsPointer}}*{{end}}r.{{.Name}})
	}
	{{- end}}

	return nil, errors.New("invalid {{.Name}}ResultOrs")
}

func (r *{{.Name}}ResultOrs) UnmarshalJSON(data []byte) error {
	// Try null
	if bytes.Equal(data, []byte("null")) {
		r.IsNull = true
		return nil
	}
	{{- range .UnionFields}}
	// Try {{.Type}}
	var {{.Name | lower}} {{if .IsArray}}[]{{end}}{{.Type}}
	if err := json.Unmarshal(data, &{{.Name | lower}}); err == nil {
		r.{{.Name}} = {{if .IsPointer}}&{{end}}{{.Name | lower}}
		return nil
	}
	{{- end}}

	return errors.New("invalid {{.Name}}ResultOrs")
}

type {{.Name}}Result struct {
	{{.Name}}ResultOrs
}
`))
		if err := tmpl.Execute(buf, resultInfo); err != nil {
			return errors.Errorf("executing result template: %w", err)
		}
	}

	// Write the generated code to file
	if err := os.WriteFile(g.outputPath, buf.Bytes(), 0644); err != nil {
		return errors.Errorf("writing output file: %w", err)
	}

	return nil
}
