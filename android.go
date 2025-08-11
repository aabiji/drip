// compile the android_utility class
// jar files are automatically included in android builds by gogio
//go:generate javac -classpath $ANDROID_HOME/platforms/android-36/android.jar -d /tmp/java_classes android_utility.java
//go:generate jar cf android_utility.jar -C /tmp/java_classes .

// TODO: write a tutorial on how to do this, explaining *why* as we go, using my class as an example

package main

/*
#cgo LDFLAGS: -llog -landroid
#include <android/log.h>
#include <android/native_window_jni.h>
#include <stdlib.h>

// Get the current env, attaching the current thread
// if it's detached. Should return JNI_OK on success.
static jint jni_GetEnvOrAttach(JavaVM *vm, JNIEnv **env, jint *attached) {
    jint res = (*vm)->GetEnv(vm, (void **)env, JNI_VERSION_1_6);
    if (res == JNI_EDETACHED) {
        res = (*vm)->AttachCurrentThread(vm, (void **)env, NULL);
		*attached = res == JNI_OK;
    }
    return res;
}

static void jni_DetachCurrent(JavaVM *vm) {
    (*vm)->DetachCurrentThread(vm);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"unsafe"

	"gioui.org/app"
	"github.com/timob/jnigi"
)

func getJNIEnv() (*jnigi.Env, func()) {
	runtime.LockOSThread()

	jvm := app.JavaVM()
	cJVM := (*C.JavaVM)(unsafe.Pointer(jvm))

	var cEnv *C.JNIEnv
	var attached C.jint = 0
	C.jni_GetEnvOrAttach(cJVM, &cEnv, &attached)

	// we're passing nil for thiz (`this` in java) since
	// we'll be calling a static method
	_, env := jnigi.UseJVM(unsafe.Pointer(jvm), unsafe.Pointer(cEnv), nil)

	cleanup := func() {
		if attached != 0 {
			C.jni_DetachCurrent(cJVM)
		}
		runtime.UnlockOSThread()
	}
	return env, cleanup
}

// since android doesn't allow us to access files and folders willy nilly,
// call the writeToDownloadsFolder static method in android_utility.java
// in order to write to the user's Downloads folder
func WriteToDownloadsFolder(
	filename string, mimetype string, contents []byte) error {
	env, cleanup := getJNIEnv()
	defer cleanup()

	// get the arguments as objects
	contextObj := jnigi.WrapJObject(app.AppContext(), "android/content/Context", false)

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
		"com/aabiji/drip/android_utility", "writeToDownloadsFolder",
		nil, // returns void
		contextObj, filenameObj, mimetypeObj, contentsObj,
	)
	return err
}

type androidLogger struct{}

func (logger androidLogger) Write(data []byte) (int, error) {
	tag := C.CString("drip-debug")
	defer C.free(unsafe.Pointer(tag))

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		msg := C.CString(line)
		C.__android_log_write(C.ANDROID_LOG_INFO, tag, msg)
		C.free(unsafe.Pointer(msg))
	}
	return len(data), nil
}

func androidCrashHandler() {
	msg := fmt.Sprintf("\n%v\n%s\n", recover(), debug.Stack())
	androidLogger{}.Write([]byte(msg))
}
