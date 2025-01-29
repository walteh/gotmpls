package generate

import (
	"strings"
	"text/template"

	_ "embed"
)

//go:embed templates/schema.go.tmpl
var schemaTemplate string

func getTemplateHelpers() template.FuncMap {
	return template.FuncMap{
		"contains": strings.Contains,
		"lastPathComponent": func(path string) string {
			parts := strings.Split(path, "/")
			return parts[len(parts)-1]
		},
		"hasSuffix": strings.HasSuffix,
		"getOrderedFieldNames": func(fields map[string]Field) []string {
			names := make([]string, 0, len(fields))
			for name := range fields {
				names = append(names, name)
			}
			return names
		},
	}
}
