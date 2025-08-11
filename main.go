package main

import (
	"fmt"
	"log"
	"runtime"
)

func main() {
	if runtime.GOOS == "android" {
		log.SetOutput(androidLogger{})
		defer androidCrashHandler()
	}

	// 08-10 19:42:29.909  3327  4102 W WifiTransportLayerUtils: getApplicationCategory - IOException com.github.drip ??
	if err := WriteToDownloadsFolder("this-works.txt", "text/plain", []byte("hello world!")); err != nil {
		panic(fmt.Sprintf("failed to write to downloads folder! %s", err.Error()))
	}
	app := NewApp()
	app.Launch()
}
