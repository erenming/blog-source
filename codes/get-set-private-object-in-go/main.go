package main

import (
	"github.com/erenming/blog-source/codes/get-set-private-object-in-go/preceiver/pa"
	"unsafe"
	_ "unsafe"
)

//go:linkname methodPrintGlobalFlag github.com/erenming/blog-source/codes/get-set-private-object-in-go/preceiver/pa.(*privateType).printGlobalFlag
func methodPrintGlobalFlag(t *main_privateType)

type main_privateType struct {
	intField    int
	stringField string
	flag        bool
}

func main() {
	// method
	t := pa.GetPrivateType()
	convertedPT := *(*main_privateType)(unsafe.Pointer(t))
	methodPrintGlobalFlag(&convertedPT)
}
