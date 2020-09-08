package main

import "C"
import (
	"fmt"
)

//PyProductEvent : Interface function for Python only dealing with JSON strings
//export PyProductEvent
func PyProductEvent(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)
	fmt.Println("Go received message:")
	fmt.Println(msg)
	return C.CString("{\"err\": false, \"errmsg\": \"\"}")
}

//PySayHello : Simple function to check the interface is working.
//export PySayHello
func PySayHello(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)
	fmt.Printf("Python says: %s\n", msg)
	return C.CString("Hello Python!")
}

func main() {}
