package lsp_test

// func TestIntegration(t *testing.T) {
// 	ctx := context.Background()

// 	server := lsp.NewServer(
// 		ast.NewDefaultPackageAnalyzer(),
// 		true,
// 	)

// 	t.Run("basic LSP flow", func(t *testing.T) {
// 		files := testFiles{
// 			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
// {{- define "header" -}}
// # Person Information
// {{- end -}}

// {{template "header"}}

// Name: {{.Name}}
// Age: {{.Age}}
// Address:
//   Street: {{.Address.Street}}
//   City: {{.Address.City}}`,
// 			"go.mod": "module test",
// 			"test.go": `
// package test

// type Person struct {
// 	Name    string
// 	Age     int
// 	Address Address
// }

// type Address struct {
// 	Street string
// 	City   string
// }`,
// 		}

// 		setup, err := setupNeovimTest(t, server, files)
// 		require.NoError(t, err, "setup should succeed")
// 		defer setup.cleanup()

// 		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

// 		// Test hover over .Name
// 		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
// 			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
// 			Position:     lsp.Position{Line: 7, Character: 8},
// 		})
// 		require.NoError(t, err, "hover request should succeed")
// 		require.NotNil(t, hoverResult, "hover result should not be nil")
// 		require.Equal(t, "**Variable**: Person.Name\n**Type**: string", hoverResult.Contents.Value)
// 		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
// 		require.Equal(t, 7, hoverResult.Range.Start.Line, "range should start on line 7")
// 		require.Equal(t, 7, hoverResult.Range.End.Line, "range should end on line 7")
// 		require.Equal(t, 8, hoverResult.Range.Start.Character, "range should start at the beginning of .Name")
// 		require.Equal(t, 13, hoverResult.Range.End.Character, "range should end at the end of .Name")

// 		// Test hover over .Age
// 		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
// 			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
// 			Position:     lsp.Position{Line: 8, Character: 7},
// 		})
// 		require.NoError(t, err, "hover request should succeed")
// 		require.NotNil(t, hoverResult, "hover result should not be nil")
// 		require.Equal(t, "**Variable**: Person.Age\n**Type**: int", hoverResult.Contents.Value)
// 		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
// 		require.Equal(t, 8, hoverResult.Range.Start.Line, "range should start on line 8")
// 		require.Equal(t, 8, hoverResult.Range.End.Line, "range should end on line 8")
// 		require.Equal(t, 7, hoverResult.Range.Start.Character, "range should start at the beginning of .Age")
// 		require.Equal(t, 11, hoverResult.Range.End.Character, "range should end at the end of .Age")

// 		// Test hover over nested field .Address.Street
// 		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
// 			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
// 			Position:     lsp.Position{Line: 10, Character: 12},
// 		})
// 		require.NoError(t, err, "hover request should succeed")
// 		require.NotNil(t, hoverResult, "hover result should not be nil")
// 		require.Equal(t, "**Variable**: Person.Address.Street\n**Type**: string", hoverResult.Contents.Value)
// 		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
// 		require.Equal(t, 10, hoverResult.Range.Start.Line, "range should start on line 10")
// 		require.Equal(t, 10, hoverResult.Range.End.Line, "range should end on line 10")
// 		require.Equal(t, 12, hoverResult.Range.Start.Character, "range should start at the beginning of .Address.Street")
// 		require.Equal(t, 26, hoverResult.Range.End.Character, "range should end at the end of .Address.Street")

// 		// Test hover over nested field .Address.City
// 		hoverResult, err = setup.requestHover(t, ctx, &lsp.HoverParams{
// 			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
// 			Position:     lsp.Position{Line: 11, Character: 10},
// 		})
// 		require.NoError(t, err, "hover request should succeed")
// 		require.NotNil(t, hoverResult, "hover result should not be nil")
// 		require.Equal(t, "**Variable**: Person.Address.City\n**Type**: string", hoverResult.Contents.Value)
// 		require.NotNil(t, hoverResult.Range, "hover range should not be nil")
// 		require.Equal(t, 11, hoverResult.Range.Start.Line, "range should start on line 11")
// 		require.Equal(t, 11, hoverResult.Range.End.Line, "range should end on line 11")
// 		require.Equal(t, 10, hoverResult.Range.Start.Character, "range should start at the beginning of .Address.City")
// 		require.Equal(t, 22, hoverResult.Range.End.Character, "range should end at the end of .Address.City")
// 	})

// 	t.Run("missing go.mod", func(t *testing.T) {
// 		files := testFiles{
// 			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
// {{ .Name }}`,
// 			"test.go": `
// package test
// type Person struct {
// 	Name string
// }`,
// 		}

// 		setup, err := setupNeovimTest(t, server, files)
// 		require.NoError(t, err, "setup should succeed")
// 		defer setup.cleanup()

// 		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

// 		// Test hover over .Name - should fail because go.mod is missing
// 		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
// 			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
// 			Position:     lsp.Position{Line: 1, Character: 3},
// 		})
// 		require.NoError(t, err, "hover request should succeed")
// 		require.Nil(t, hoverResult, "hover should return nil when go.mod is missing")
// 	})

// 	t.Run("invalid go.mod", func(t *testing.T) {
// 		files := testFiles{
// 			"test.tmpl": `{{- /*gotype: test.Person*/ -}}
// {{ .Name }}`,
// 			"go.mod": "invalid go.mod content",
// 			"test.go": `
// package test
// type Person struct {
// 	Name string
// }`,
// 		}

// 		setup, err := setupNeovimTest(t, server, files)
// 		require.NoError(t, err, "setup should succeed")
// 		defer setup.cleanup()

// 		testFile := filepath.Join(setup.tmpDir, "test.tmpl")

// 		// Test hover over .Name - should fail because go.mod is invalid
// 		hoverResult, err := setup.requestHover(t, ctx, &lsp.HoverParams{
// 			TextDocument: lsp.TextDocumentIdentifier{URI: "file://" + testFile},
// 			Position:     lsp.Position{Line: 1, Character: 3},
// 		})
// 		require.NoError(t, err, "hover request should succeed")
// 		require.Nil(t, hoverResult, "hover should return nil when go.mod is invalid")
// 	})
// }
