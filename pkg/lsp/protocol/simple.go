package protocol

func NewHoverParams(uri DocumentURI, position Position) *HoverParams {
	return &HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}
}
