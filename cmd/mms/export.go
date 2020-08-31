package main

import "C"

//PyPostEvent : Interface function for Python only dealing with JSON strings
//export PyPostEvent
func PyPostEvent(msg string) string {
	return "{\"err\": False, \"errmsg\": \"\"}"
}

//PyHello : Simple function to check the interface is working.
//export PyHello
func PyHello(x int) int {
	return 2 * x
}
