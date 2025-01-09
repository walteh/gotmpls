package debug

import (
	"context"
	"os"

	"github.com/rs/zerolog"
)

// Printf prints debug messages to stderr if GOTMPL_DEBUG is set
func Printf(format string, args ...interface{}) {
	if os.Getenv("GOTMPL_DEBUG") == "" {
		return
	}

	logger := zerolog.New(os.Stderr).With().
		Str("component", "debug").
		Timestamp().
		Logger()

	ctx := logger.WithContext(context.Background())
	zerolog.Ctx(ctx).Debug().Msgf(format, args...)
}
