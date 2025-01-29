package diff

import (
	"reflect"
	"testing"

	"github.com/tj/assert"
)

func AssertEqual[T any](t *testing.T, expected, actual T) {
	de := reflect.DeepEqual(expected, actual)
	if !de {
		str := "\n\n============= TYPE COMPARISON START =============\n\n"

		str += TypedDiff(expected, actual) + "\n\n"

		str += "\n============= TYPE COMPARISON END ===============\n\n"
		t.Log("type comparison report:\n" + str)
		assert.True(t, de, "type mismatch")
	}
}
