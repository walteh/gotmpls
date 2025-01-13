// Package hover provides functionality for generating hover information in the LSP server.
package hover

// // LSPRange represents a range in a document for LSP
// type LSPRange struct {
// 	Start LSPPosition `json:"start"`
// 	End   LSPPosition `json:"end"`
// }

// // LSPPosition represents a position in a document for LSP
// type LSPPosition struct {
// 	Line      int `json:"line"`
// 	Character int `json:"character"`
// }

// // LSPHover represents a hover response for LSP
// type LSPHover struct {
// 	Contents LSPMarkupContent `json:"contents"`
// 	Range    *LSPRange        `json:"range,omitempty"`
// }

// LSPMarkupContent represents markup content for LSP
// type LSPMarkupContent struct {
// 	Kind  string `json:"kind"`
// 	Value string `json:"value"`
// }

// // ToLSPHover converts a HoverInfo to an LSP hover response
// func (h *HoverInfo) ToLSPHover() *LSPHover {
// 	if h == nil {
// 		return nil
// 	}

// 	return &LSPHover{
// 		Contents: LSPMarkupContent{
// 			Kind:  "markdown",
// 			Value: h.Content,
// 		},
// 		Range: rangeToLSPRange(h.Range),
// 	}
// }
