package lsp_test

import (
	"context"
	"os"
	"testing"

	"github.com/walteh/go-tmpl-typer/gen/mockery"
	"github.com/walteh/go-tmpl-typer/pkg/diagnostic"
	"github.com/walteh/go-tmpl-typer/pkg/lsp"
)

func TestServer_Start(t *testing.T) {
	if os.Getenv("LSP_TEST_SERVER") != "1" {
		t.Skip("skipping LSP server test; LSP_TEST_SERVER not set")
	}

	// Create a new server with mock implementations
	mockParser := mockery.NewMockTemplateParser_parser(t)
	mockValidator := mockery.NewMockValidator_types(t)
	mockAnalyzer := mockery.NewMockPackageAnalyzer_ast(t)

	server := lsp.NewServer(
		mockParser,
		mockValidator,
		mockAnalyzer,
		diagnostic.NewDefaultGenerator(),
		true,
	)

	// Start the server using stdin/stdout
	ctx := context.Background()
	err := server.Start(ctx, os.Stdin, os.Stdout)
	if err != nil {
		t.Fatalf("failed to serve LSP: %v", err)
	}
}
