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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

type targetError struct {
	message             string
	file                string
	function            string
	missingTemplateInfo bool // true if it was created from something other than the t.errorf / t.errorfNoPanic method
}

var (
	missingValueForIf = targetError{message: "missing value for if", file: "parse.go", function: "checkPipeline"}
	unexpectedEOF     = targetError{message: "unexpected EOF", file: "parse.go", function: "itemList"}
	unexpectedEnd     = targetError{message: "unexpected {{end}}", file: "parse.go", function: "itemList"}
)

var targetErrors = []targetError{
	missingValueForIf,
	unexpectedEOF,
}

var checkedErrors = make(map[targetError]int)

func init() {
	for _, err := range targetErrors {
		checkedErrors[err] = 0
	}
	// special case because this error will never have something behind it
	checkedErrors[unexpectedEOF] = 1
}

type aggregationTest struct {
	name           string
	template       string
	expectedErrors []targetError
}

var aggregationTests = []aggregationTest{

	{
		// Template structure:
		// ┌──────────┐
		// │   Hello  │ (TextNode)
		// ├──────────┤
		// │  {{if    │ (Missing if condition)
		// └──────────┘
		name:     "single_error_missing_end",
		template: "Hello {{if}}",
		expectedErrors: []targetError{
			missingValueForIf,
			unexpectedEOF,
		},
	},
	{
		// ┌──────────┐
		// │   Hello  │ (TextNode)
		// ├──────────┤
		// │  {{if    │ (Missing if condition)
		// ├──────────┤
		// │  {{if    │ (Missing if condition)
		// ├──────────┤
		// │  {{if    │ (Missing if condition)
		// └──────────┘
		name:     "if_end_missing_content_extra_end",
		template: "Hello {{if}} {{end}} {{end}}",
		expectedErrors: []targetError{
			missingValueForIf,
			unexpectedEnd,
		},
	},
	{
		name:     "if_end_missing_with_comments",
		template: "Hello {{if}} {{/* comment */}} {{end}} {{end}}",
		expectedErrors: []targetError{
			missingValueForIf,
			unexpectedEnd,
		},
	},
	{
		name:     "missing_value_if",
		template: "Hello {{if}}{{if}}{{if}}",
		expectedErrors: []targetError{
			missingValueForIf,
			missingValueForIf,
			missingValueForIf,
			unexpectedEOF,
		},
	},
}

func TestErrorAggregation(t *testing.T) {
	for _, test := range aggregationTests {
		t.Run(test.name, func(t *testing.T) {
			tr := New(test.name)
			tr.Mode = ParseComments
			_, err := tr.Parse(test.template, "", "", make(map[string]*Tree))
			if len(test.expectedErrors) == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err) // for backwards compatibility so all the other tests don't fail
			}
			// we need to rebuild the errors because the error type is different
			// other will check the parser errors directly, but here we don't care about the type
			returnedErrors := tr.Errors()
			rebuiltErrors := make([]string, len(returnedErrors))
			for i, err := range returnedErrors {
				str := err.Error()
				if strings.Contains(str, "template:") {
					// remove the template: whatever:number:
					str = strings.TrimPrefix(str, "template: ")
					str = strings.TrimPrefix(str, test.name+":")
					str = strings.Split(str, ":")[1]
					str = strings.TrimSpace(str)
				}
				rebuiltErrors[i] = str
			}
			rebuiltExpectedErrors := make([]string, len(test.expectedErrors))
			for i, err := range test.expectedErrors {
				rebuiltExpectedErrors[i] = err.message
			}
			require.Equal(t, rebuiltExpectedErrors, rebuiltErrors)
			for i, err := range test.expectedErrors {
				if i < len(returnedErrors)-1 {
					// we don't check the last error because nothing was aggregated behind it
					checkedErrors[err]++
				}
			}
		})
	}
	t.Run("all_errors_processed", func(t *testing.T) {
		for targetError, count := range checkedErrors {
			assert.GreaterOrEqual(t, count, 1, "error not processed: [message='%s' file='%s' function='%s']", targetError.message, targetError.file, targetError.function)
		}
	})
}
