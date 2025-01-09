package main

import (
	"context"
	"os"
	"runtime"

	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("app", "go-tmpl-typer").
		Str("os", runtime.GOOS).
		Logger()

	ctx := logger.WithContext(context.Background())
	zerolog.Ctx(ctx).Info().Str("status", "starting").Msg("application initialized")
}
