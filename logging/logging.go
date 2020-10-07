package logging

import (
	"fmt"
	"log"
)

type logEntryType string

const (
	entryTypeInfo    = "[ INFO ]"
	entryTypeWarning = "[ WARN ]"
	entryTypeError   = "[ ERR  ]"
)

func Infof(format string, v ...interface{}) {
	printToLog(entryTypeInfo, fmt.Sprintf(format, v...))
}

func Info(msg string) {
	printToLog(entryTypeInfo, msg)
}

func Warningf(format string, v ...interface{}) {
	printToLog(entryTypeWarning, fmt.Sprintf(format, v...))
}

func Warning(msg string) {
	printToLog(entryTypeWarning, msg)
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	printToLog(entryTypeError, fmt.Sprintf(format, v...))
}

func Error(msg string) {
	printToLog(entryTypeError, msg)
}

func printToLog(entryType logEntryType, msg string) {
	log.Printf("%s %s", entryType, msg)
}
