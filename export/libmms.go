package main

import "C"
import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/metno/go-mms/pkg/mms"
)

//PyProductEvent : Interface function for Python only dealing with JSON strings
//export PyProductEvent
func PyProductEvent(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)
	fmt.Println("Go received message:")
	fmt.Println(msg)

	var productEvent mms.ProductEvent
	json.Unmarshal([]byte(msg), &productEvent)
	productEvent.CreatedAt = time.Now()

	hubs := mms.ListProductionHubs()
	err := mms.MakeProductEvent(hubs, &productEvent)
	if err != nil {
		return C.CString(fmt.Sprintf("{\"err\": true, \"errmsg\": \"%s\"}", err.Error()))
	} else {
		return C.CString("{\"err\": false, \"errmsg\": \"\"}")
	}
}

//PySayHello : Simple function to check the interface is working.
//export PySayHello
func PySayHello(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)
	fmt.Printf("Python says: %s\n", msg)
	return C.CString("Hello Python!")
}

func main() {}
