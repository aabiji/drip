package main

import "runtime"

func main() {
	if runtime.GOOS == "android" {
		defer AndroidPanicLogger()
	}

	app := NewApp()
	app.Launch()
}
