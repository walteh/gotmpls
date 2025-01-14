package integration

import (
	"context"
	"testing"

	"github.com/walteh/go-tmpl-typer/pkg/lsp/protocol"
)

type IntegrationTestRunner interface {
	Hover(t *testing.T, ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error)
}
