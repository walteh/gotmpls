package diff

import (
	"strings"

	"github.com/k0kubun/pp/v3"
	"github.com/kylelemons/godebug/diff"
)

func DiffExportedOnly[T any](want T, got T) string {
	printer := pp.New()
	printer.SetExportedOnly(true)
	printer.SetColoringEnabled(false)
	abc := diff.Diff(printer.Sprint(got), printer.Sprint(want))
	if abc == "" {
		return ""
	}
	str := "\n\n"
	str += "to convert ACTUAL ⏩️ EXPECTED:\n\n"
	str += "add:    ➕\n"
	str += "remove: ➖\n"
	str += "\n"
	str += strings.ReplaceAll(strings.ReplaceAll(abc, "\n-", "\n➖"), "\n+", "\n➕")

	return str
}
