package generate

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"
)

// TemplateData holds all the data needed for code generation
type TemplateData struct {
	PackageName string
	Aliases     []Field
	Structs     []struct {
		Name         string
		Description  string
		Fields       map[string]Field
		GenerateCode bool
	}
}

// Output generates code and writes to w.
func Output(w io.Writer, g *Generator, pkg string) error {
	// Parse template with helper functions
	tmpl, err := template.New("schema").Funcs(getTemplateHelpers()).Parse(schemaTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Prepare template data
	data := TemplateData{
		PackageName: pkg, // No longer need to clean package name as we handle it in template
		Aliases:     make([]Field, 0, len(g.Aliases)),
		Structs: make([]struct {
			Name         string
			Description  string
			Fields       map[string]Field
			GenerateCode bool
		}, 0, len(g.Structs)),
	}

	// Add aliases in sorted order
	for _, k := range getOrderedFieldNames(g.Aliases) {
		data.Aliases = append(data.Aliases, g.Aliases[k])
	}

	// Add structs in sorted order
	for _, k := range getOrderedStructNames(g.Structs) {
		s := g.Structs[k]
		data.Structs = append(data.Structs, struct {
			Name         string
			Description  string
			Fields       map[string]Field
			GenerateCode bool
		}{
			Name:         s.Name,
			Description:  s.Description,
			Fields:       s.Fields,
			GenerateCode: s.GenerateCode,
		})
	}

	// Execute template
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

// Helper functions
func getOrderedFieldNames(fields map[string]Field) []string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getOrderedStructNames(m map[string]Struct) []string {
	keys := make([]string, len(m))
	idx := 0
	for k := range m {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)
	return keys
}

func cleanPackageName(pkg string) string {
	pkg = strings.Replace(pkg, ".", "", -1)
	pkg = strings.Replace(pkg, "-", "", -1)
	return pkg
}
