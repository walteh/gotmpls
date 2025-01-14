package protocol

func NewHoverParams(uri string, position Position) *HoverParams {
	return &HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: DocumentURI(uri)},
			Position:     position,
		},
	}
}
