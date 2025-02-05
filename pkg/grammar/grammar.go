package grammar

import (
	"archive/tar"
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	textmategrammarsthemes "github.com/walteh/gotmpls/gen/git-repo-tarballs/textmate-grammars-themes"
	"github.com/walteh/gotmpls/pkg/targz"
	"gitlab.com/tozd/go/errors"
)

// Grammar represents a TextMate grammar definition
type Grammar struct {
	Name               string              `json:"name,omitempty"`
	ScopeName          string              `json:"scopeName"`
	FileTypes          []string            `json:"fileTypes,omitempty"`
	FoldingStartMarker string              `json:"foldingStartMarker,omitempty"`
	FoldingStopMarker  string              `json:"foldingStopMarker,omitempty"`
	FirstLineMatch     string              `json:"firstLineMatch,omitempty"`
	Patterns           []Pattern           `json:"patterns"`
	Repository         map[string]*Pattern `json:"repository,omitempty"`
}

// Pattern represents a single pattern in a TextMate grammar
type Pattern struct {
	// Common fields
	Include string `json:"include,omitempty"`
	Match   string `json:"match,omitempty"`
	Name    string `json:"name,omitempty"`
	Comment string `json:"comment,omitempty"`

	// Begin/end fields
	Begin            string              `json:"begin,omitempty"`
	End              string              `json:"end,omitempty"`
	BeginCaptures    map[string]*Capture `json:"-"` // Custom unmarshaling
	EndCaptures      map[string]*Capture `json:"-"` // Custom unmarshaling
	Captures         map[string]*Capture `json:"-"` // Custom unmarshaling
	RawBeginCaptures json.RawMessage     `json:"beginCaptures,omitempty"`
	RawEndCaptures   json.RawMessage     `json:"endCaptures,omitempty"`
	RawCaptures      json.RawMessage     `json:"captures,omitempty"`

	// Nested patterns
	Patterns []Pattern `json:"patterns,omitempty"`
}

// Capture represents a capture group in a pattern
type Capture struct {
	Name     string    `json:"name,omitempty"`
	Patterns []Pattern `json:"patterns,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface for Pattern
func (p *Pattern) UnmarshalJSON(data []byte) error {
	type Alias Pattern
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Initialize maps
	p.BeginCaptures = make(map[string]*Capture)
	p.EndCaptures = make(map[string]*Capture)
	p.Captures = make(map[string]*Capture)

	// Handle beginCaptures
	if p.RawBeginCaptures != nil {
		if err := p.unmarshalCaptures(p.RawBeginCaptures, p.BeginCaptures); err != nil {
			return errors.Errorf("unmarshaling beginCaptures: %w", err)
		}
	}

	// Handle endCaptures
	if p.RawEndCaptures != nil {
		if err := p.unmarshalCaptures(p.RawEndCaptures, p.EndCaptures); err != nil {
			return errors.Errorf("unmarshaling endCaptures: %w", err)
		}
	}

	// Handle captures
	if p.RawCaptures != nil {
		if err := p.unmarshalCaptures(p.RawCaptures, p.Captures); err != nil {
			return errors.Errorf("unmarshaling captures: %w", err)
		}
	}

	return nil
}

// unmarshalCaptures handles both object and array formats
func (p *Pattern) unmarshalCaptures(data []byte, target map[string]*Capture) error {
	// Try unmarshaling as a map first
	var mapData map[string]json.RawMessage
	if err := json.Unmarshal(data, &mapData); err == nil {
		for k, v := range mapData {
			var capture Capture
			if err := json.Unmarshal(v, &capture); err != nil {
				return errors.Errorf("unmarshaling capture %s: %w", k, err)
			}
			target[k] = &capture
		}
		return nil
	}

	// If map fails, try array
	var arrayData []json.RawMessage
	if err := json.Unmarshal(data, &arrayData); err != nil {
		return err
	}

	// Convert array to map
	for i, v := range arrayData {
		var capture Capture
		if err := json.Unmarshal(v, &capture); err != nil {
			return errors.Errorf("unmarshaling capture %d: %w", i, err)
		}
		target[strconv.Itoa(i)] = &capture
	}

	return nil
}

// CaptureValue represents a capture value which can be either a string or a Capture object
type CaptureValue struct {
	Name     string    `json:"name,omitempty"`
	Patterns []Pattern `json:"patterns,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (c *CaptureValue) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		c.Name = str
		return nil
	}

	// If that fails, try to unmarshal as a Capture object
	type captureAlias CaptureValue
	var capture captureAlias
	if err := json.Unmarshal(data, &capture); err != nil {
		return err
	}
	*c = CaptureValue(capture)
	return nil
}

// Store manages a collection of TextMate grammars
type Store struct {
	grammars map[string]*Grammar
}

// NewStore creates a new grammar store and loads embedded grammars
func NewStore(ctx context.Context) (*Store, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("creating new grammar store")

	s := &Store{
		grammars: make(map[string]*Grammar),
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

		var grammar Grammar
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

	var grammar Grammar
	if err := json.Unmarshal(data, &grammar); err != nil {
		return errors.Errorf("unmarshaling custom grammar: %w", err)
	}

	s.grammars[name] = &grammar
	return nil
}

// GetGrammar retrieves a grammar by name
func (s *Store) GetGrammar(name string) (*Grammar, error) {
	grammar, ok := s.grammars[name]
	if !ok {
		return nil, errors.Errorf("grammar not found: %s", name)
	}
	return grammar, nil
}
