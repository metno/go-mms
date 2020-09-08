package main

import "C"
import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/metno/go-mms/pkg/mms"
)

//PyProductEvent is an interface function for Python for posting a prodction event.
//export PyProductEvent
func PyProductEvent(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)

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

//PySayHello is a simple function to check the interface is working.
//export PySayHello
func PySayHello(cMsg *C.char) *C.char {
	msg := C.GoString(cMsg)
	fmt.Printf("Python says: %s\n", msg)
	return C.CString("Hello Python!")
}

func main() {}
