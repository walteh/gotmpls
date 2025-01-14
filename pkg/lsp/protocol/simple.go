package protocol

func NewHoverParams(uri DocumentURI, position Position) *HoverParams {
	return &HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}
}

func NewPosition(line, character int) Position {
	return Position{
		Line:      uint32(line),
		Character: uint32(character),
	}
}

func NewRange(start, end Position) Range {
	return Range{
		Start: start,
		End:   end,
	}
}

func NewLocation(uri DocumentURI, rng Range) Location {
	return Location{
		URI:   uri,
		Range: rng,
	}
}

func NewDiagnostic(rng Range, message string, severity DiagnosticSeverity) Diagnostic {
	return Diagnostic{
		Range:    rng,
		Message:  message,
		Severity: severity,
	}
}

func NewTextDocumentItem(uri DocumentURI, languageID LanguageKind, version int, text string) TextDocumentItem {
	return TextDocumentItem{
		URI:        uri,
		LanguageID: languageID,
		Version:    int32(version),
		Text:       text,
	}
}

func NewVersionedTextDocumentIdentifier(uri DocumentURI, version int) VersionedTextDocumentIdentifier {
	return VersionedTextDocumentIdentifier{
		TextDocumentIdentifier: TextDocumentIdentifier{URI: uri},
		Version:                int32(version),
	}
}

func NewTextDocumentContentChangeEvent(text string) *TextDocumentContentChangeEvent {
	return &TextDocumentContentChangeEvent{
		Text: text,
	}
}

func NewMarkupContent(kind MarkupKind, value string) MarkupContent {
	return MarkupContent{
		Kind:  kind,
		Value: value,
	}
}
