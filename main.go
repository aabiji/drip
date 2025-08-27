package main

import (
	"fmt"
	"log"
	"runtime/debug"
)

type OSBridge interface {
	// write a file to disk
	WriteFile(filename string, mimetype string, contents []byte) error
	// io.Writer implementation for the custom logger
	Write(data []byte) (int, error)
}

func main() {
	bridge := NewOSBridge()
	log.SetOutput(bridge)

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("%v\n%s\n", recover(), debug.Stack())
			bridge.Write([]byte(msg))
		}
	}()

	//if err := bridge.WriteFile("this-works.txt", "text/plain", []byte("hello world!")); err != nil {
	//	panic(fmt.Sprintf("failed to write to downloads folder! %s", err.Error()))
	//}

	app := NewApp(&bridge)
	app.Launch()
}
