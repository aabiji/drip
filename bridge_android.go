//go:build android

// compile the android_utility class
// jar files are automatically included in android builds by gogio
//go:generate javac -classpath $ANDROID_HOME/platforms/android-36/android.jar -d /tmp/java_classes android_utility.java
//go:generate jar cf android_utility.jar -C /tmp/java_classes .

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

type AndroidBridge struct {
	context *jnigi.ObjectRef // android.content.Content
	env     *jnigi.Env
	cleanup func()
}

func NewOSBridge() OSBridge {
	env, cleanup := getJNIEnv()
	context := jnigi.WrapJObject(app.AppContext(), "android/content/Context", false)
	return &AndroidBridge{context, env, cleanup}
}

func (b *AndroidBridge) getDownloadsFolderPath() (string, error) {
	strObj, err := b.env.NewObject("java/lang/String")
	if err != nil {
		return "", err
	}

	err = b.env.CallStaticMethod(
		"com/aabiji/drip/android_utility", "getDownloadsFolderPath", strObj)
	if err != nil {
		return "", err
	}

	var bytes []byte
	if err := strObj.CallMethod(b.env, "getBytes", &bytes, b.env.GetUTF8String()); err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (b *AndroidBridge) WriteFile(
	filename string, mimetype string, contents []byte) error {
	// get the arguments as objects
	basePath, err := b.getDownloadsFolderPath()
	if err != nil {
		return err
	}
	fmt.Println(basePath)
	basePathObj, err := b.env.NewObject("java/lang/String", []byte(basePath))
	if err != nil {
		return err
	}

	filenameObj, err := b.env.NewObject("java/lang/String", []byte(filename))
	if err != nil {
		return err
	}

	mimetypeObj, err := b.env.NewObject("java/lang/String", []byte(mimetype))
	if err != nil {
		return err
	}

	contentsObj := b.env.NewByteArrayFromSlice(contents)

	err = b.env.CallStaticMethod(
		"com/aabiji/drip/android_utility", "writeToPath",
		nil, // returns void
		b.context, contentsObj, basePathObj, filenameObj, mimetypeObj,
	)
	return err
}

func (b AndroidBridge) Write(data []byte) (int, error) {
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
