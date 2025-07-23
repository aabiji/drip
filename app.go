package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/aabiji/drip/p2p"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Settings struct {
	Theme             string `json:"theme"`
	TrustPeers        bool   `json:"trustPeers"`
	ShowNotifications bool   `json:"showNotifications"`
	DownloadFolder    string `json:"downloadFolder"`
}

func loadSettings(configPath string) Settings {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	defaultFolder := path.Join(home, "Downloads")

	settings := Settings{
		Theme:             "light",
		TrustPeers:        true,
		ShowNotifications: true,
		DownloadFolder:    defaultFolder,
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

	if len(strings.TrimSpace(settings.DownloadFolder)) == 0 {
		settings.DownloadFolder = defaultFolder
	}
	return settings
}

type App struct {
	ctx    context.Context
	events chan p2p.Message

	finder     *p2p.PeerFinder
	downloader *p2p.Downloader

	configPath string
	settings   Settings
}

func NewApp() *App {
	events := make(chan p2p.Message, 25)
	finder := p2p.NewPeerFinder(true, events)
	configPath := "drip-settings.json"
	settings := loadSettings(configPath)
	app := &App{
		events:     events,
		finder:     &finder,
		configPath: configPath,
		settings:   settings,
	}
	app.downloader = p2p.NewDownloader(
		&app.settings.DownloadFolder,
		app.AuthorizePeer, app.SignalSessionCompletion)
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// launch the peer finder service
	go func() {
		if err := a.finder.Run(a.ctx, a.downloader); err != nil {
			panic(err)
		}
	}()

	// launch the event emitter service
	go func() {
		for event := range a.events {
			runtime.EventsEmit(a.ctx, event.MessageType, event)
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	// save the settings
	jsonData, err := json.Marshal(a.settings)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(a.configPath, jsonData, 0644); err != nil {
		panic(err)
	}

	// shutdown services
	a.downloader.Close()
	for _, peer := range a.finder.Peers {
		peer.Close()
	}
	close(a.events)
}

func (a *App) AuthorizePeer(peer string) bool {
	result, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Transfer authorization",
		Message:       fmt.Sprintf("Allow %s to send you some files?", peer),
		DefaultButton: "Yes",
		Buttons:       []string{"Yes", "No"},
	})
	if err != nil {
		panic(err)
	}
	return result == "Yes"
}

func (a *App) SignalSessionCompletion(peer string, numReceived int) {
	_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Transfer received",
		Message: fmt.Sprintf("Received %d files from %s", numReceived, peer),
		Buttons: []string{"Ok"},
	})
	if err != nil {
		panic(err)
	}
}

// the following functions are exported to the frontend
func (a *App) GetPeers() []string { return a.finder.GetConnectedPeers() }

func (a *App) GetSettings() Settings          { return a.settings }
func (a *App) SaveSettings(settings Settings) { a.settings = settings }

func (a *App) RequestSessionAuth(info p2p.SessionInfo) error {
	msg := p2p.NewMessage(p2p.SESSION_INFO, info)
	for _, peerId := range info.Recipients {
		peer, exists := a.finder.Peers[peerId]
		if exists && peer.Connected() {
			peer.Webrtc.QueueMessage(msg)
		}
	}
	return nil
}

func (a *App) SendFileChunk(chunk p2p.TransferChunk) error {
	peer, exists := a.finder.Peers[chunk.Recipient]
	if !exists || !peer.Connected() {
		return fmt.Errorf("%s has disconnected", chunk.Recipient)
	}

	msg := p2p.NewMessage(p2p.TRANSFER_CHUNK, chunk)
	a.finder.Peers[chunk.Recipient].Webrtc.QueueMessage(msg)
	return nil
}

func (a *App) CancelSession(signal p2p.SessionCancel) error {
	msg := p2p.NewMessage(p2p.SESSION_CANCEL, signal)
	for _, peerId := range signal.Recipients {
		peer, exists := a.finder.Peers[peerId]
		if exists {
			peer.Webrtc.QueueMessage(msg)
		}
	}
	return nil
}

func (a *App) SelectDowloadFolder() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	path, err := runtime.OpenDirectoryDialog(a.ctx,
		runtime.OpenDialogOptions{DefaultDirectory: home})
	if len(path) == 0 || err != nil { // user didn't select anything
		return a.settings.DownloadFolder
	}

	a.settings.DownloadFolder = path
	return path
}
