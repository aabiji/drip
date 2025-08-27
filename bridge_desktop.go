//go:build !android && !ios

package main

import (
	"fmt"
	"os"
)

type DesktopBridge struct{} // the default

func NewOSBridge() OSBridge { return &DesktopBridge{} }

func (b *DesktopBridge) WriteFile(filename string, mimetype string, contents []byte) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	_, err = file.Write(contents)
	return err
}

func (b *DesktopBridge) Write(data []byte) (int, error) {
	fmt.Println(string(data))
	return len(data), nil
}
