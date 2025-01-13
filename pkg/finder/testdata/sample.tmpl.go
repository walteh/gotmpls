// {{- /*gotype: github.com/example/types.Config */ -}}
package templates

type Data struct {
	Name string
	Age  int
}

// {{define "main"}}
// Hello {{.Name}}! You are {{.Age}} years old.
// {{end}}
