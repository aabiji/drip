package main

import (
	"encoding/json"
	"github.com/kirsle/configdir"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	DownloadFolder    string
	TrustPeers        bool
	ShowNotifications bool
	LightMode         bool
}

func configPath() string {
	configPath := configdir.LocalConfig("drip")
	err := configdir.MakePath(configPath)
	if err != nil {
		panic(err)
	}
	return filepath.Join(configPath, "settings.json")
}

func saveSettings(s Settings) {
	jsonData, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(configPath(), jsonData, 0644); err != nil {
		panic(err)
	}
}

func loadSettings() Settings {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	defaultFolder := filepath.Join(home, "Downloads")

	settings := Settings{
		LightMode:         true,
		TrustPeers:        true,
		ShowNotifications: true,
		DownloadFolder:    defaultFolder,
	}

	file, err := os.Open(configPath())
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

	if len(strings.TrimSpace(settings.DownloadFolder)) == 0 {
		settings.DownloadFolder = defaultFolder
	}
	return settings
}
