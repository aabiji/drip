package main

import (
	"log"
	"runtime"
)

func main() {
	if runtime.GOOS == "android" {
		log.SetOutput(androidLogger{})
		defer androidCrashHandler()
	}
	app := NewApp()
	app.Launch()
}
