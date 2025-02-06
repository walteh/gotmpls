# LSP Protocol Code Generator Requirements

## 🎯 Goal

Generate clean, idiomatic Go code from the LSP protocol specification (metaModel.json)

## 📝 Example: textDocument/implementation Request

### Input (from metaModel.json)

```json
{
	"method": "textDocument/implementation",
	"typeName": "ImplementationRequest",
	"result": {
		"kind": "or",
		"items": [
			{ "kind": "reference", "name": "Definition" },
			{
				"kind": "array",
				"element": { "kind": "reference", "name": "DefinitionLink" }
			},
			{ "kind": "base", "name": "null" }
		]
	},
	"params": {
		"kind": "reference",
		"name": "ImplementationParams"
	},
	"documentation": "A request to resolve the implementation locations of a symbol at a given text document position."
}
```

### Expected Output (Go code)

```go
// ImplementationRequest represents a request to resolve the implementation locations of a symbol
// at a given text document position.
type ImplementationParams struct {
    TextDocumentURI string `json:"textDocumentURI"`
    Position        Position `json:"position"`
}

type ImplementationRequest struct {
	ImplementationParams
}

type ImplementationResult struct {
	ImplementationResultOrs
}

// OrDefinitionDefinitionLinkNull represents a union type of Definition, []DefinitionLink, or null
type ImplementationResultOrs struct {
    Definition      *Definition
    DefinitionLinks []DefinitionLink
    IsNull         bool
}

func (r ImplementationResultOrs) MarshalJSON() ([]byte, error) {
	if r.IsNull {
		return json.Marshal(nil)
	}

	if r.Definition != nil {
		return json.Marshal(r.Definition)
	}

	if r.DefinitionLinks != nil {
		return json.Marshal(r.DefinitionLinks)
	}

	return nil, errors.New("invalid implementation result")
}

func (r *ImplementationResultOrs) UnmarshalJSON(data []byte) error {
	// try to unmarshal as nil
	if bytes.Equal(data, []byte("null")) {
		r.IsNull = true
		return nil
	}

	// try to unmarshal as Definition
	if err := json.Unmarshal(data, &r.Definition); err == nil {
		return nil
	}


	// try to unmarshal as DefinitionLinks
	if err := json.Unmarshal(data, &r.DefinitionLinks); err == nil {
		return nil
	}

	return errors.New("invalid implementation result")
}
```

## 🔑 Key Requirements

1. Type Generation:

    - Convert LSP types to Go structs
    - Handle union types with clear naming
    - Generate proper JSON tags
    - Include documentation comments

2. Naming Conventions:

    - Request types: `{Name}Request`
    - Union types: `Or{Type1}{Type2}...`
    - Method constants: `{TypeName}Method`

3. Type Handling:

    - Base types map to Go primitives (string, int32, bool)
    - Arrays use slice syntax `[]Type`
    - References use the referenced type name
    - Unions become structs with optional fields

4. Documentation:
    - Include original LSP documentation
    - Add ASCII art separators for visual clarity
    - Document union type possibilities

## 🧪 Testing Strategy

1. Create test with a real request from metaModel.json
2. Parse the request into our model
3. Generate Go code
4. Verify the output matches our expectations

## 📋 Implementation Steps

1. Parse metaModel.json into vscodemetamodel.MetaModel
2. Extract a single request to start with
3. Generate the request type and its dependencies
4. Add JSON marshaling support
5. Add documentation
6. Verify with tests

## 🎨 Code Style

```go
// Example of our preferred code style
type ExampleRequest struct {
    // Original LSP documentation goes here
    // Additional context if needed

    // Required fields first
    Method string `json:"method"`

    // Optional fields with omitempty
    Params *ExampleParams `json:"params,omitempty"`
}

// Constants at the end
const ExampleRequestMethod = "example/method"
```
