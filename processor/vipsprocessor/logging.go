package vipsprocessor

// #include <glib.h>
// #include "logging.h"
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

type LoggingHandlerFunction func(messageDomain string, messageLevel LogLevel, message string)

func SetLogging(handler LoggingHandlerFunction, verbosity LogLevel) {
	if handler != nil {
		currentLoggingHandlerFunction = handler
	}
	currentLoggingVerbosity = verbosity
}

func noopLoggingHandler(_ string, _ LogLevel, _ string) {
}

func log(domain string, level LogLevel, message string) {
	if level <= currentLoggingVerbosity {
		currentLoggingHandlerFunction(domain, level, message)
	}
}

func enableLogging() {
	C.set_logging_handler()
}

func disableLogging() {
	C.unset_logging_handler()
}
