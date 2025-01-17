///////////////////////////////////////////////////////
// Custom tests for modifications to text/template/parse
//
// 🎯 Test Philosophy:
// - Each test should be tiny and focused on ONE thing
// - Tests should be heavily documented with ASCII art showing the template structure
// - Use emojis to make it clear what aspect we're testing
//
// 🔍 Test Categories:
// - 📝 Basic parsing
// - ❌ Error handling
// - 🔍 Position tracking
// - ✂️ Trim markers
// - 🔄 Delimiters
///////////////////////////////////////////////////////

package parse

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// verifies that we collect all parsing errors
//
// Template structure:
// ┌──────────┐
// │   Hello  │ (TextNode)
// ├──────────┤
// │  {{if   │ (Missing if condition)
// └──────────┘
func TestErrorAggregationOne(t *testing.T) {
	// Given a template with a malformed if statement
	template := "Hello {{if}}"

	// When we parse it
	trees, err := Parse("test", template, "{{", "}}", nil)
	require.Error(t, err, "parsing template with errors should return error")

	// Then we should get a tree with errors
	tree := trees["test"]
	require.NotNil(t, tree, "tree should not be nil")

	// And the tree should have errors
	errs := tree.Errors()
	require.NotNil(t, errs, "errors should not be nil")
	require.NotEmpty(t, errs, "should have at least one error")

	// And the error should indicate missing if condition
	assert.Contains(t, errs[0].Error(), "missing value", "error should mention missing value")
}

// TestErrorAggregationTwo verifies that we collect multiple errors in a single parse
//
// the primary purpose of this test is to ensure that the error aggregation works
//
// Template structure:
// ┌──────────┐
// │   Hello  │ (TextNode)
// ├──────────┤
// │  {{if   │ (Missing if condition)
// ├──────────┤
// │  {{end  │ (Extra end)
// └──────────┘
func TestErrorAggregationTwo(t *testing.T) {
	// Given a template with multiple errors
	template := "Hello {{if}} {{end}} {{end}}" // Missing if condition and extra end

	expectedErrors := []error{
		errors.New("template: test:1: missing value for if"),
		errors.New("template: test:1: unexpected {{end}}"),
	}

	// When we parse it
	trees, err := Parse("test", template, "{{", "}}", nil)
	require.EqualError(t, err, expectedErrors[0].Error()) // for backwards compatibility so all the other tests don't fail

	// Then we should get a tree with errors
	tree := trees["test"]
	require.NotNil(t, tree, "tree should not be nil")

	// And the tree should have multiple errors
	errs := tree.Errors()
	require.NotNil(t, errs, "errors should not be nil")
	require.ElementsMatch(t, expectedErrors, errs, "errors should match expected")
}

// TestErrorAggregationTwo verifies that we collect multiple errors in a single parse
//
// the primary purpose of this test is to ensure that the error aggregation works with the "missing value" error in parse.go
//
// Template structure:
// ┌──────────┐
// │   Hello  │ (TextNode)
// ├──────────┤
// │  {{if   │ (Missing if condition)
// ├──────────┤
// │  {{if   │ (Missing if condition)
// ├──────────┤
// │  {{if   │ (Missing if condition)
// └──────────┘
func TestErrorAggregationMissingValue(t *testing.T) {
	// Given a template with a missing value
	template := "Hello {{if}}{{if}}{{if}}"

	expectedErrors := []error{
		errors.New("template: test:1: missing value for if"),
		errors.New("template: test:1: missing value for if"),
		errors.New("template: test:1: missing value for if"),
		errors.New("template: test:1: unexpected EOF"),
	}

	// When we parse it
	trees, err := Parse("test", template, "{{", "}}", nil)
	require.Error(t, err, "parsing template with errors should return error")

	// Then we should get a tree with errors
	tree := trees["test"]
	require.NotNil(t, tree, "tree should not be nil")

	// And the tree should have errors
	errs := tree.Errors()
	require.NotNil(t, errs, "errors should not be nil")
	require.ElementsMatch(t, expectedErrors, errs, "errors should match expected")
}
