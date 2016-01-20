package log

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
)

const (
	flags = log.Ldate | log.Ltime | log.Lmicroseconds
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
		linfo:  log.New(stdout, "[INFO ", flags),
		lerr:   log.New(stderr, "[ERRO ", flags),
		lfatal: log.New(stderr, "[FATA ", flags),
		lpanic: log.New(stderr, "[PANI ", flags),
	}
}

func suffix() string {
	if _, file, line, ok := runtime.Caller(2); ok {
		return fmt.Sprintf("%s:%d]", path.Base(file), line)
	}
	return ""
}

// Info log.
func (l *impl) Info(v ...interface{}) {
	l.linfo.Println(suffix(), fmt.Sprint(v...))
}

// Infof log.
func (l *impl) Infof(format string, v ...interface{}) {
	l.linfo.Println(suffix(), fmt.Sprintf(format, v...))
}

// Error log.
func (l *impl) Error(v ...interface{}) {
	l.lerr.Println(suffix(), fmt.Sprint(v...))
}

// Errorf log.
func (l *impl) Errorf(format string, v ...interface{}) {
	l.lerr.Println(suffix(), fmt.Sprintf(format, v...))
}

// Fatal log.
func (l *impl) Fatal(v ...interface{}) {
	l.lfatal.Println(suffix(), fmt.Sprint(v...))
}

// Fatalf log.
func (l *impl) Fatalf(format string, v ...interface{}) {
	l.lfatal.Println(suffix(), fmt.Sprintf(format, v...))
}

// Panic log.
func (l *impl) Panic(v ...interface{}) {
	l.lpanic.Println(suffix(), fmt.Sprint(v...))
	panic("")
}

// Panicf log.
func (l *impl) Panicf(format string, v ...interface{}) {
	l.lpanic.Println(suffix(), fmt.Sprintf(format, v...))
	panic("")
}

var (
	DefaultLogger = NewLogger(os.Stdout, os.Stderr)
	base          = DefaultLogger.(*impl)
)

func init() {
	base = NewLogger(os.Stdout, os.Stderr).(*impl)
}

// Info log.
func Info(v ...interface{}) {
	base.linfo.Println(suffix(), fmt.Sprint(v...))
}

// Infof log.
func Infof(format string, v ...interface{}) {
	base.linfo.Println(suffix(), fmt.Sprintf(format, v...))
}

// Error log.
func Error(v ...interface{}) {
	base.lerr.Println(suffix(), fmt.Sprint(v...))
}

// Errorf log.
func Errorf(format string, v ...interface{}) {
	base.lerr.Println(suffix(), fmt.Sprintf(format, v...))
}

// Fatal log.
func Fatal(v ...interface{}) {
	base.lfatal.Fatalln(suffix(), fmt.Sprint(v...))
}

// Fatalf log.
func Fatalf(format string, v ...interface{}) {
	base.lfatal.Fatalln(suffix(), fmt.Sprintf(format, v...))
}

// Panic log.
func Panic(v ...interface{}) {
	base.lpanic.Println(suffix(), fmt.Sprint(v...))
	panic("")
}

// Panicf log.
func Panicf(format string, v ...interface{}) {
	base.lpanic.Println(suffix(), fmt.Sprintf(format, v...))
	panic("")
}
