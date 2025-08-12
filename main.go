package main

import (
	"fmt"
	"log"
)

func main() {
	bridge := NewAndroidBridge()
	if bridge != nil {
		log.SetOutput(*bridge)
		defer androidCrashHandler()
	}

	if err := bridge.WriteFile("this-works.txt", "text/plain", []byte("hello world!")); err != nil {
		panic(fmt.Sprintf("failed to write to downloads folder! %s", err.Error()))
	}

	app := NewApp(bridge)
	app.Launch()
}
