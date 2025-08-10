package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/widget"
)

type Settings struct {
	DownloadPath string
	TrustPeers   widget.Bool
	NotifyUser   widget.Bool
	DarkMode     widget.Bool
	path         string
}

func saveSettings(s Settings) {
	jsonData, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(s.path, jsonData, 0644); err != nil {
		panic(err)
	}
}

func loadSettings() Settings {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	defaultFolder := filepath.Join(home, "Downloads")

	base, err := app.DataDir()
	if err != nil {
		panic(err)
	}
	configPath := filepath.Join(base, "settings.json")

	settings := Settings{
		DarkMode:     widget.Bool{Value: false},
		TrustPeers:   widget.Bool{Value: true},
		NotifyUser:   widget.Bool{Value: true},
		DownloadPath: defaultFolder,
		path:         configPath,
	}

	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		return settings // return defaults if there's no settings file
	} else if err != nil {
		panic(err)
	}
	defer file.Close()

	contents, _ := io.ReadAll(file)
	if err := json.Unmarshal(contents, &settings); err != nil {
		panic(err)
	}

	if len(strings.TrimSpace(settings.DownloadPath)) == 0 {
		settings.DownloadPath = defaultFolder
	}
	return settings
}
