package integration

import (
	"context"
	"testing"
	"time"

	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

type IntegrationTestRunner interface {
	Hover(t *testing.T, ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error)
	CheckDiagnostics(t *testing.T, uri protocol.DocumentURI, expectedDiagnostics []protocol.Diagnostic, timeout time.Duration) error
	ApplyEditWithSave(t *testing.T, uri protocol.DocumentURI, newContent string) error
	ApplyEditWithoutSave(t *testing.T, uri protocol.DocumentURI, newContent string) error
}
