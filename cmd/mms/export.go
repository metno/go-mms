package main

import "C"

//export PyPostEvent
func PyPostEvent(msg string) string {
	return "{\"err\": False, \"errmsg\": \"\"}"
}
