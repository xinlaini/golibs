package xlog

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
)

const (
	flags = log.Ldate | log.Ltime | log.Lmicroseconds
)

var (
	colorConsole = flag.Bool("color_console", true, "Whether to print colored console log")
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

type pretty struct {
	linfo   *log.Logger
	lerr    *log.Logger
	lfatal  *log.Logger
	lpanic  *log.Logger
	colored bool
}

// NewLogger creates new logger using stdout and stderr files.
func NewLogger(stdout, stderr *os.File) Logger {
	return &pretty{
		linfo:  log.New(stdout, "[INFO ", flags),
		lerr:   log.New(stderr, "[ERRO ", flags),
		lfatal: log.New(stderr, "[FATA ", flags),
		lpanic: log.New(stderr, "[PANI ", flags),
	}
}

func NewConsoleLogger() Logger {
	if !*colorConsole {
		return NewLogger(os.Stdout, os.Stderr)
	}

	return &pretty{
		linfo:   log.New(os.Stdout, "\x1b[1;32m[", flags),
		lerr:    log.New(os.Stderr, "\x1b[1;31m[", flags),
		lfatal:  log.New(os.Stderr, "\x1b[1;35m[", flags),
		lpanic:  log.New(os.Stderr, "\x1b[1;33m[", flags),
		colored: true,
	}
}

func (l *pretty) suffix() string {
	if _, file, line, ok := runtime.Caller(2); ok {
		if l.colored {
			return fmt.Sprintf("%s:%d]\x1b[0m", path.Base(file), line)
		}
		return fmt.Sprintf("%s:%d]", path.Base(file), line)
	}
	return ""
}

// Info log.
func (l *pretty) Info(v ...interface{}) {
	l.linfo.Println(l.suffix(), fmt.Sprint(v...))
}

// Infof log.
func (l *pretty) Infof(format string, v ...interface{}) {
	l.linfo.Println(l.suffix(), fmt.Sprintf(format, v...))
}

// Error log.
func (l *pretty) Error(v ...interface{}) {
	l.lerr.Println(l.suffix(), fmt.Sprint(v...))
}

// Errorf log.
func (l *pretty) Errorf(format string, v ...interface{}) {
	l.lerr.Println(l.suffix(), fmt.Sprintf(format, v...))
}

// Fatal log.
func (l *pretty) Fatal(v ...interface{}) {
	l.lfatal.Fatalln(l.suffix(), fmt.Sprint(v...))
}

// Fatalf log.
func (l *pretty) Fatalf(format string, v ...interface{}) {
	l.lfatal.Fatalln(l.suffix(), fmt.Sprintf(format, v...))
}

// Panic log.
func (l *pretty) Panic(v ...interface{}) {
	l.lpanic.Println(l.suffix(), fmt.Sprint(v...))
	panic("")
}

// Panicf log.
func (l *pretty) Panicf(format string, v ...interface{}) {
	l.lpanic.Println(l.suffix(), fmt.Sprintf(format, v...))
	panic("")
}

type plain struct {
	lout *log.Logger
	lerr *log.Logger
}

func NewPlainLogger() Logger {
	return &plain{
		lout: log.New(os.Stdout, "", 0),
		lerr: log.New(os.Stderr, "", 0),
	}
}

func NewNilLogger() Logger {
	return &plain{
		lout: log.New(ioutil.Discard, "", 0),
		lerr: log.New(ioutil.Discard, "", 0),
	}
}

// Info log.
func (pl *plain) Info(v ...interface{}) {
	pl.lout.Println(v...)
}

// Infof log.
func (pl *plain) Infof(format string, v ...interface{}) {
	pl.lout.Println(fmt.Sprintf(format, v...))
}

// Error log.
func (pl *plain) Error(v ...interface{}) {
	pl.lerr.Println(v...)
}

// Errorf log.
func (pl *plain) Errorf(format string, v ...interface{}) {
	pl.lerr.Println(fmt.Sprintf(format, v...))
}

// Fatal log.
func (pl *plain) Fatal(v ...interface{}) {
	pl.lerr.Fatalln(v...)
}

// Fatalf log.
func (pl *plain) Fatalf(format string, v ...interface{}) {
	pl.lerr.Fatalln(fmt.Sprintf(format, v...))
}

// Panic log.
func (pl *plain) Panic(v ...interface{}) {
	pl.lerr.Println(v...)
	panic("")
}

// Panicf log.
func (pl *plain) Panicf(format string, v ...interface{}) {
	pl.lerr.Println(fmt.Sprintf(format, v...))
	panic("")
}
