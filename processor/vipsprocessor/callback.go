package vipsprocessor

import "C"

//export goLoggingHandler
func goLoggingHandler(messageDomain *C.char, messageLevel C.int, message *C.char) {
	log(C.GoString(messageDomain), LogLevel(messageLevel), C.GoString(message))
}
