package debug

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
)

func hackGetCallerSkipFrameCount(e *zerolog.Event) int {
	// Access the unexported skipCaller field
	v := reflect.ValueOf(e).Elem() // Get the value of the pointer
	field := v.FieldByName("skipFrame")

	if field.IsValid() && field.CanAddr() {
		// Use unsafe to bypass field access restrictions
		return int(field.Int())
	}

	return 0
}

type CustomTimeHook struct {
	WithColor bool
	Format    string
}

func (t CustomTimeHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	if t.Format == "" {
		// milisecond precision with no timezone

		str := time.Now().Format("2006-01-02T15:04:05.0000Z")
		// strWithPadding := fmt.Sprintf("%-26s", str)
		e.Str("time", str)
	} else {
		e.Str("time", time.Now().Format(t.Format))
	}
}

type CustomCallerHook struct {
	WithColor bool
}

func (c CustomCallerHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {

	pc, file, line, ok := runtime.Caller(hackGetCallerSkipFrameCount(e) + 3)
	if !ok {
		return
	}

	funcd := runtime.FuncForPC(pc)

	pkg, _ := GetPackageAndFuncFromFuncName(funcd.Name())

	e.Str("caller", FormatCaller(pkg, file, line, c.WithColor))

	return

}

// func ZeroLogCallerMarshalFunc(pc uintptr, file string, line int) string {

// }

func GetPackageAndFuncFromFuncName(pc string) (pkg, function string) {
	funcName := pc
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}

	firstDot := strings.IndexByte(funcName[lastSlash:], '.') + lastSlash

	pkg = funcName[:firstDot]
	fname := funcName[firstDot+1:]

	if strings.Contains(pkg, ".(") {
		splt := strings.Split(pkg, ".(")
		pkg = splt[0]
		fname = "(" + splt[1] + "." + fname
	}

	// pkg = strings.TrimPrefix(pkg, currentGoPackage+"/")

	return pkg, fname
}

func FormatCaller(pkg, path string, number int, colorize bool) string {
	// pkg = filepath.Base(pkg)
	p := FileNameOfPath(path)
	if colorize {
		p = color.New(color.Bold).Sprint(p)
		num := color.New(color.FgHiRed, color.Bold).Sprintf("%d", number)
		sep := color.New(color.Faint).Sprint(":")

		return fmt.Sprintf("%s%s%s%s%s", pkg, sep, p, sep, num)
	}

	return fmt.Sprintf("%s:%s:%d", pkg, p, number)
}

func FileNameOfPath(path string) string {
	tot := strings.Split(path, "/")
	if len(tot) > 1 {
		return tot[len(tot)-1]
	}

	return path
}
