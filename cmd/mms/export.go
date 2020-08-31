package main

import "C"
import "fmt"

//PyPostEvent : Interface function for Python only dealing with JSON strings
//export PyPostEvent
func PyPostEvent(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)
	fmt.Println("Go received message:")
	fmt.Println(msg)
	return C.CString("{\"err\": false, \"errmsg\": \"\"}")
}

//SayHello : Simple function to check the interface is working.
//export SayHello
func SayHello(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)
	fmt.Printf("Python says: %s\n", msg)
	return C.CString("Hello Python!")
}
