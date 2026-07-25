package service

import (
	"unsafe"

	"github.com/tidwall/gjson"
)

func parseRawJSONView(raw []byte) gjson.Result {
	if len(raw) == 0 {
		return gjson.Result{}
	}
	return gjson.Parse(*(*string)(unsafe.Pointer(&raw)))
}
