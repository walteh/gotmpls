package lsp

import (
	"io"
	"os"
	"sync"

	"github.com/walteh/gotmpls/pkg/lsp/protocol"
	"github.com/walteh/gotmpls/pkg/parser"
)

// Document represents a text document with its metadata
type Document struct {
	URI        string
	LanguageID protocol.LanguageKind
	Version    int32
	Content    string
	AST        *parser.ParsedTemplateFile
}

// DocumentManager handles document operations
type DocumentManager struct {
	store *sync.Map // map[string]*Document
}

func NewDocumentManager() *DocumentManager {
	return &DocumentManager{
		store: &sync.Map{},
	}
}

func (m *DocumentManager) GetNoFallback(uri protocol.DocumentURI) (*Document, bool) {
	normalizedURI := normalizeURI(string(uri))
	content, ok := m.store.Load(normalizedURI)
	if content == nil {
		return nil, ok
	} else {
		return content.(*Document), ok
	}
}

func (m *DocumentManager) Get(uri protocol.DocumentURI) (*Document, bool) {
	normalizedURI := normalizeURI(string(uri))
	content, ok := m.store.Load(normalizedURI)
	if !ok {
		// Try with the original URI as fallback
		content, ok = m.store.Load("file://" + uri)
	}
	if !ok {
		// try filesystem
		file, err := os.Open(normalizedURI)
		if err != nil {
			return nil, false
		}
		defer file.Close()
		contentz, err := io.ReadAll(file)
		if err != nil {
			return nil, false
		}
		doc := &Document{
			URI:     normalizedURI,
			Content: string(contentz),
		}
		m.store.Store(normalizedURI, doc)
		return doc, true
	}

	doc, ok := content.(*Document)
	return doc, ok
}

func (m *DocumentManager) Store(uri protocol.DocumentURI, doc *Document) {
	normalizedURI := normalizeURI(string(uri))
	m.store.Store(normalizedURI, doc)
}

func (m *DocumentManager) Delete(uri string) {
	normalizedURI := normalizeURI(uri)
	m.store.Delete(normalizedURI)
}
