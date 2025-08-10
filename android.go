// compile the android_utility class
// jar files are automatically included in android builds by gogio
//go:generate javac -classpath $ANDROID_HOME/platforms/android-36/android.jar -d /tmp/java_classes android_utility.java
//go:generate jar cf android_utility.jar -C /tmp/java_classes .

package main

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime/debug"
	"strings"
	"unsafe"

	"gioui.org/app"
	"github.com/timob/jnigi"
)

type androidLogger struct{}

func (logger androidLogger) Write(data []byte) (int, error) {
	tag := C.CString("drip-debug")
	defer C.free(unsafe.Pointer(tag))

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if len(line) > 0 {
			msg := C.CString(line)
			C.__android_log_write(C.ANDROID_LOG_INFO, tag, msg)
			C.free(unsafe.Pointer(msg))
		}
	}
	return len(data), nil
}

func androidCrashHandler() {
	msg := fmt.Sprintf("\n%v\n%s\n", recover(), debug.Stack())
	androidLogger{}.Write([]byte(msg))
}

// since android doesn't allow us to access files and folders willy nilly,
// call the writeToDownloadsFolder static method in android_utility.java
// in order to write to the user's Downloads folder
func WriteToDownloadsFolder(
	filename string, mimetype string, contents []byte) error {

	if app.JavaVM() == 0 {
		panic("javaJvm is nil")
	}
	if app.AppContext() == 0 {
		panic("appContext is nil")
	}

	// use the existing android jvm
	jvm := (*jnigi.JVM)(unsafe.Pointer(app.JavaVM()))
	if jvm == nil {
		panic("jvm's nil")
	}
	env := jvm.AttachCurrentThread()
	defer jvm.DetachCurrentThread(env)

	// get the arguments as objects
	contextObj := jnigi.WrapJObject(app.AppContext(), "android/content/Context", false)
	if contextObj == nil {
		panic("context is nil")
	}

	filenameObj, err := env.NewObject("java/lang/String", []byte(filename))
	if err != nil {
		return err
	}

	mimetypeObj, err := env.NewObject("java/lang/String", []byte(mimetype))
	if err != nil {
		return err
	}

	contentsObj := env.NewByteArrayFromSlice(contents).GetObject()

	// call the function
	err = env.CallStaticMethod(
		"org/aabiji/drip/android_utility", "writeToDownloadsFolder",
		nil, // returns void
		contextObj, filenameObj, mimetypeObj, contentsObj,
	)
	return err
}
