//go:build android

package main

/*
#cgo LDFLAGS: -llog

#include <android/log.h>
#include <stdlib.h>

void android_log(int prio, const char* tag, const char* msg) {
    __android_log_write(prio, tag, msg);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func AndroidPanicLogger() {
	if r := recover(); r != nil {
		ctag := C.CString("drip-debug")
		cmsg := C.CString(fmt.Sprintf("Crashed! %s", r.(string)))
		C.android_log(C.ANDROID_LOG_FATAL, ctag, cmsg)
		C.free(unsafe.Pointer(ctag))
		C.free(unsafe.Pointer(cmsg))
	}
}
