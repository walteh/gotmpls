/*
Package semtok provides semantic token support for the Go template language server.

🎨 Semantic Tokens Overview:
---------------------------
Semantic tokens provide enhanced syntax highlighting and visual understanding
of code by identifying specific language elements and their roles. This package
bridges our template parser with the LSP protocol's semantic token requirements.

Architecture:
------------

	Template Text                 LSP Server
	     |                            |
	     v                            v
	+----------+    tokens     +-------------+
	| @parser  | -----------> |   @semtok   |
	+----------+              +-------------+
	     |                          |
	AST Nodes                       |
	     |                    +-----+------+
	     |                    |            |
	     |               Full File    Range-based
	     |               Tokens      Tokens
	     v
	Position Info

🔍 Main Components:
-----------------
1. Token Provider Interface
  - Converts AST nodes to semantic tokens
  - Maps template elements to LSP token types
  - Handles position calculations

2. Token Types
  - variable    (template variables)
  - function    (template functions)
  - keyword     (template keywords like if, range)
  - operator    (.|, etc.)
  - string      (string literals)
  - comment     (template comments)

3. Token Modifiers
  - declaration (first occurrence)
  - readonly    (constants)
  - static      (global items)

📝 Implementation Plan:
--------------------
TODO(@semtok): Phase 1 - Core Infrastructure
- [ ] Define token types and modifiers
- [ ] Create TokenProvider interface
- [ ] Implement basic position mapping

TODO(@semtok): Phase 2 - Parser Integration
- [ ] Add AST node visitors
- [ ] Implement token generation for each node type
- [ ] Add position calculation helpers

TODO(@semtok): Phase 3 - LSP Integration
- [ ] Implement LSP semantic token protocol
- [ ] Add range-based token support
- [ ] Add full file token support

TODO(@semtok): Phase 4 - Testing & Validation
- [ ] Add unit tests for each token type
- [ ] Add integration tests with LSP
- [ ] Add performance benchmarks

🔗 Related Packages:
------------------
- @parser: Provides AST nodes and position information
- @lsp: Consumes semantic tokens for client communication

Example Usage:
-------------

	tokens, err := semtok.GetTokensForFile(ctx, content)
	if err != nil {
	    return err
	}
	// Use tokens in LSP response...

Note: This package is designed to be thread-safe and context-aware,
making it suitable for concurrent LSP requests.
*/
package semtok

/*
Implementation Details & Notes:
-----------------------------

AST -> Semantic Token Mapping:

    AST Node         ->   Semantic Token
    --------              --------------
    Variable        ->    variable
    Function        ->    function
    Text           ->    string
    Action         ->    keyword
    Pipeline       ->    operator
    Comment        ->    comment

Position Handling:
-----------------
We need to be careful with position mapping because template
positions might need to be adjusted for LSP client expectations.

    Template Pos    ->    LSP Position
    ------------          -------------
    Line:Col            Line:Character

TODO(@parser): Ensure position information includes:
- [ ] Start position (line, character)
- [ ] End position (line, character)
- [ ] Content length
*/
