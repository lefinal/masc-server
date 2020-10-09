package logging

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"log"
)

type logEntryType string

const (
	entryTypeInfo    = "[ INFO ]"
	entryTypeWarning = "[ WARN ]"
	entryTypeError   = "[ ERR  ]"
)

// Logger logs stuff.
type Logger struct {
	subject string
}

// NewLogger creates a new Logger which uses the given subject.
func NewLogger(subject string) Logger {
	return Logger{
		subject: subject,
	}
}

// GeneralLogger creates a new Logger which uses no subject.
func GeneralLogger() Logger {
	return Logger{}
}

func (l Logger) Infof(format string, v ...interface{}) {
	l.printToLog(entryTypeInfo, fmt.Sprintf(format, v...))
}

func (l Logger) Info(msg string) {
	l.printToLog(entryTypeInfo, msg)
}

func (l Logger) Warningf(format string, v ...interface{}) {
	l.printToLog(entryTypeWarning, fmt.Sprintf(format, v...))
}

func (l Logger) Warning(msg string) {
	l.printToLog(entryTypeWarning, msg)
}

func (l Logger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func (l Logger) Fatal(err error) {
	log.Fatal(err)
}

func (l Logger) Errorf(format string, v ...interface{}) {
	l.printToLog(entryTypeError, fmt.Sprintf(format, v...))
}

func (l Logger) Error(msg string) {
	l.printToLog(entryTypeError, msg)
}

func (l Logger) MascError(err *errors.MascError) {
	l.Error(err.Error())
}

func (l Logger) printToLog(entryType logEntryType, msg string) {
	var subjectInsert string
	if l.subject != "" {
		subjectInsert = fmt.Sprintf(" [%s] ", l.subject)
	}
	log.Printf("%s%s %s", entryType, subjectInsert, msg)
}
