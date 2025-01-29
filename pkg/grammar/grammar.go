package grammar

import (
	"archive/tar"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	textmategrammarsthemes "github.com/walteh/gotmpls/gen/git-repo-tarballs/textmate-grammars-themes"
	tmlanguage "github.com/walteh/gotmpls/gen/jsonschema/tmlanguage"
	"github.com/walteh/gotmpls/pkg/targz"
	"gitlab.com/tozd/go/errors"
)

// Grammar represents a TextMate grammar definition

// Store manages a collection of TextMate grammars
type Store struct {
	grammars map[string]*tmlanguage.Grammar
}

// NewStore creates a new grammar store and loads embedded grammars
func NewStore(ctx context.Context) (*Store, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("creating new grammar store")

	s := &Store{
		grammars: make(map[string]*tmlanguage.Grammar),
	}

	store, err := targz.LoadTarGzWithOptions(textmategrammarsthemes.Data, targz.LoadOptions{
		Filter: func(header *tar.Header) bool {
			return strings.Contains(header.Name, "packages/tm-grammars/raw") &&
				strings.HasSuffix(header.Name, ".json")
		},
	})
	if err != nil {
		return nil, errors.Errorf("loading tarball: %w", err)
	}

	// Log all loaded grammars
	for scope, grammarc := range store.Files {
		// parse the grammars as json

		var grammar tmlanguage.Grammar
		if err := json.Unmarshal(grammarc, &grammar); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Str("scope", scope).RawJSON("grammar", grammarc).Msg("unmarshaling grammar")
			return nil, errors.Errorf("unmarshaling grammar %s: %w", scope, err)
		}
		name := strings.TrimSuffix(filepath.Base(scope), ".json")
		s.grammars[name] = &grammar
	}

	return s, nil
}

// LoadCustomGrammar loads a custom grammar from JSON data
func (s *Store) LoadCustomGrammar(ctx context.Context, name string, data []byte) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("name", name).Msg("loading custom grammar")

	var grammar tmlanguage.Grammar
	if err := json.Unmarshal(data, &grammar); err != nil {
		return errors.Errorf("unmarshaling custom grammar: %w", err)
	}

	s.grammars[name] = &grammar
	return nil
}

// GetGrammar retrieves a grammar by name
func (s *Store) GetGrammar(name string) (*tmlanguage.Grammar, error) {
	grammar, ok := s.grammars[name]
	if !ok {
		return nil, errors.Errorf("grammar not found: %s", name)
	}
	return grammar, nil
}
