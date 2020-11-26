/*
  Copyright 2020 MET Norway

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

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

	err := mms.MakeProductEvent("", &productEvent)
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
