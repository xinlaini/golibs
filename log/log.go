package log

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
)

const (
	flags     = log.Ldate | log.Ltime | log.Lmicroseconds
	infoSign  = "[\U0001f3b8  "
	errorSign = "[\U0001f6ab  "
	fatalSign = "[\U0001f480  "
	panicSign = "[\U0001f4a5  "
)

var (
	DefaultLogger = NewLogger(os.Stdout, os.Stderr)
	console       = newConsoleLogger()
	colorConsole  = flag.Bool("color_console", true, "Whether to print colored console log")
)

// Logger interface.
type Logger interface {
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
}

type impl struct {
	linfo  *log.Logger
	lerr   *log.Logger
	lfatal *log.Logger
	lpanic *log.Logger
}

// NewLogger creates new logger using stdout and stderr files.
func NewLogger(stdout, stderr *os.File) Logger {
	return &impl{
		linfo:  log.New(stdout, infoSign, flags),
		lerr:   log.New(stderr, errorSign, flags),
		lfatal: log.New(stderr, fatalSign, flags),
		lpanic: log.New(stderr, panicSign, flags),
	}
}

func newConsoleLogger() *impl {
	if !*colorConsole {
		return NewLogger(os.Stdout, os.Stderr).(*impl)
	}

	return &impl{
		linfo:  log.New(os.Stdout, "\x1b[1;32m"+infoSign, flags),
		lerr:   log.New(os.Stderr, "\x1b[1;31m"+errorSign, flags),
		lfatal: log.New(os.Stderr, "\x1b[1;35m"+fatalSign, flags),
		lpanic: log.New(os.Stderr, "\x1b[1;33m"+panicSign, flags),
	}
}

func suffix(resetColor bool) string {
	if _, file, line, ok := runtime.Caller(2); ok {
		if resetColor {
			return fmt.Sprintf("%s:%d]\x1b[0m", path.Base(file), line)
		}
		return fmt.Sprintf("%s:%d]", path.Base(file), line)
	}
	return ""
}

// Info log.
func (l *impl) Info(v ...interface{}) {
	l.linfo.Println(suffix(false), fmt.Sprint(v...))
}

// Infof log.
func (l *impl) Infof(format string, v ...interface{}) {
	l.linfo.Println(suffix(false), fmt.Sprintf(format, v...))
}

// Error log.
func (l *impl) Error(v ...interface{}) {
	l.lerr.Println(suffix(false), fmt.Sprint(v...))
}

// Errorf log.
func (l *impl) Errorf(format string, v ...interface{}) {
	l.lerr.Println(suffix(false), fmt.Sprintf(format, v...))
}

// Fatal log.
func (l *impl) Fatal(v ...interface{}) {
	l.lfatal.Println(suffix(false), fmt.Sprint(v...))
}

// Fatalf log.
func (l *impl) Fatalf(format string, v ...interface{}) {
	l.lfatal.Println(suffix(false), fmt.Sprintf(format, v...))
}

// Panic log.
func (l *impl) Panic(v ...interface{}) {
	l.lpanic.Println(suffix(false), fmt.Sprint(v...))
	panic("")
}

// Panicf log.
func (l *impl) Panicf(format string, v ...interface{}) {
	l.lpanic.Println(suffix(false), fmt.Sprintf(format, v...))
	panic("")
}

// Info log.
func Info(v ...interface{}) {
	console.linfo.Println(suffix(*colorConsole), fmt.Sprint(v...))
}

// Infof log.
func Infof(format string, v ...interface{}) {
	console.linfo.Println(suffix(*colorConsole), fmt.Sprintf(format, v...))
}

// Error log.
func Error(v ...interface{}) {
	console.lerr.Println(suffix(*colorConsole), fmt.Sprint(v...))
}

// Errorf log.
func Errorf(format string, v ...interface{}) {
	console.lerr.Println(suffix(*colorConsole), fmt.Sprintf(format, v...))
}

// Fatal log.
func Fatal(v ...interface{}) {
	console.lfatal.Fatalln(suffix(*colorConsole), fmt.Sprint(v...))
}

// Fatalf log.
func Fatalf(format string, v ...interface{}) {
	console.lfatal.Fatalln(suffix(*colorConsole), fmt.Sprintf(format, v...))
}

// Panic log.
func Panic(v ...interface{}) {
	console.lpanic.Println(suffix(*colorConsole), fmt.Sprint(v...))
	panic("")
}

// Panicf log.
func Panicf(format string, v ...interface{}) {
	console.lpanic.Println(suffix(*colorConsole), fmt.Sprintf(format, v...))
	panic("")
}
