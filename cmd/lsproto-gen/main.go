package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/walteh/gotmpls/cmd/lsproto-gen/generator"
	"github.com/walteh/gotmpls/gen/jsonschema/go/vscodemetamodel"
	"gitlab.com/tozd/go/errors"
)

// TODO(lsproto): 🧪 This is a proof of concept for generating LSP protocol code
// We will expand this to handle all types and generate proper code
// Current focus is on union types with proper JSON marshaling

func main() {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout)
	ctx = logger.WithContext(ctx)

	if err := run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to run generator")
	}
}

func run(ctx context.Context) error {
	// Read metamodel file
	metaModelPath := filepath.Join("gen", "jsonschema", "json", "vscodemetamodel", "metaModel.json")
	metaModelBytes, err := os.ReadFile(metaModelPath)
	if err != nil {
		return errors.Errorf("reading metamodel file: %w", err)
	}

	// Parse metamodel
	var model vscodemetamodel.MetaModel
	if err := json.Unmarshal(metaModelBytes, &model); err != nil {
		return errors.Errorf("parsing metamodel: %w", err)
	}

	// Create generator
	gen := generator.NewFileGenerator(&model)

	// Generate files
	files, err := gen.GenerateFiles(ctx, "lsproto")
	if err != nil {
		return errors.Errorf("generating files: %w", err)
	}

	// Write files
	for _, file := range files {
		// Create output directory
		outDir := filepath.Join("gen", "lsproto")
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return errors.Errorf("creating output directory: %w", err)
		}

		// Write file
		outPath := filepath.Join(outDir, file.Path)
		if err := os.WriteFile(outPath, []byte(file.Contents), 0644); err != nil {
			return errors.Errorf("writing file %s: %w", outPath, err)
		}

		logger := zerolog.Ctx(ctx)
		logger.Info().
			Str("path", outPath).
			Msg("generated file")
	}

	fmt.Println(`
	🎉 Code generation complete! 
	
	Generated files:
	└── gen/
	    └── lsproto/
	        └── types.go

	Next steps:
	1. Review the generated code
	2. Run tests
	3. Add more LSP types to the model
	`)

	return nil
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}
