package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Template
	}{
		{
			name:  "simple_text",
			input: "hello world",
			expected: &Template{
				Nodes: []*Node{
					{
						Text: &TextNode{
							Text: "hello world",
						},
					},
				},
			},
		},
		{
			name:  "simple_action",
			input: "{{.Field}}",
			expected: &Template{
				Nodes: []*Node{
					{
						Action: &ActionNode{
							Pipeline: &Pipeline{
								First: &Command{
									Argument: &Argument{
										Field: &Field{
											Dot:   true,
											Names: []string{"Field"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "pipeline",
			input: "{{.Field | upper}}",
			expected: &Template{
				Nodes: []*Node{
					{
						Action: &ActionNode{
							Pipeline: &Pipeline{
								First: &Command{
									Argument: &Argument{
										Field: &Field{
											Dot:   true,
											Names: []string{"Field"},
										},
									},
								},
								Rest: []*Command{
									{
										FuncCall: &FuncCall{
											Name: "upper",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "comment",
			input: "{{/* hello */}}",
			expected: &Template{
				Nodes: []*Node{
					{
						Comment: &CommentNode{
							Text: "{{/* hello */}}",
						},
					},
				},
			},
		},
		{
			name:  "mixed_content",
			input: "Hello {{.Name}}, welcome to {{.Place}}!",
			expected: &Template{
				Nodes: []*Node{
					{
						Text: &TextNode{
							Text: "Hello ",
						},
					},
					{
						Action: &ActionNode{
							Pipeline: &Pipeline{
								First: &Command{
									Argument: &Argument{
										Field: &Field{
											Dot:   true,
											Names: []string{"Name"},
										},
									},
								},
							},
						},
					},
					{
						Text: &TextNode{
							Text: ", welcome to ",
						},
					},
					{
						Action: &ActionNode{
							Pipeline: &Pipeline{
								First: &Command{
									Argument: &Argument{
										Field: &Field{
											Dot:   true,
											Names: []string{"Place"},
										},
									},
								},
							},
						},
					},
					{
						Text: &TextNode{
							Text: "!",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := Parser.ParseString("", tt.input)
			require.NoError(t, err, "parsing should succeed")
			require.Equal(t, tt.expected, ast, "AST should match expected structure")
		})
	}
}
