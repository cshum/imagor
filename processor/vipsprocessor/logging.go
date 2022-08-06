package vipsprocessor

// #include <glib.h>
import "C"

type LogLevel int

const (
	LogLevelError    LogLevel = C.G_LOG_LEVEL_ERROR
	LogLevelCritical LogLevel = C.G_LOG_LEVEL_CRITICAL
	LogLevelWarning  LogLevel = C.G_LOG_LEVEL_WARNING
	LogLevelMessage  LogLevel = C.G_LOG_LEVEL_MESSAGE
	LogLevelInfo     LogLevel = C.G_LOG_LEVEL_INFO
	LogLevelDebug    LogLevel = C.G_LOG_LEVEL_DEBUG
)

var (
	currentLoggingHandlerFunction = noopLoggingHandler
	currentLoggingVerbosity       LogLevel
)

//export govipsLoggingHandler
func govipsLoggingHandler(messageDomain *C.char, messageLevel C.int, message *C.char) {
	govipsLog(C.GoString(messageDomain), LogLevel(messageLevel), C.GoString(message))
}

type LoggingHandlerFunction func(messageDomain string, messageLevel LogLevel, message string)

func loggingSettings(handler LoggingHandlerFunction, verbosity LogLevel) {
	if handler != nil {
		currentLoggingHandlerFunction = handler
	}
	currentLoggingVerbosity = verbosity
}

func noopLoggingHandler(_ string, _ LogLevel, _ string) {
}

func govipsLog(domain string, level LogLevel, message string) {
	if level <= currentLoggingVerbosity {
		currentLoggingHandlerFunction(domain, level, message)
	}
}
