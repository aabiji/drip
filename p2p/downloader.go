package p2p

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strings"

	"github.com/edsrzf/mmap-go"
)

// file transfer message types
type TransferInfo struct {
	TransferId string `json:"transferId"`
	Recipient  string `json:"recipient"`
	FileName   string `json:"name"`
	FileSize   int64  `json:"size"`
}

type FileChunk struct {
	TransferId string  `json:"transferId"`
	Recipient  string  `json:"recipient"`
	Data       []uint8 `json:"data"`
	Offset     int64   `json:"offset"`
}

type TransferState struct {
	TransferId     string `json:"transferId"`
	AmountReceived int64  `json:"amountReceived"`
	file           mmap.MMap
}

type Downloader struct {
	Transfers      map[string]TransferInfo   `json:"transfers"`
	States         map[string]*TransferState `json:"states"`
	DownloadFolder string                    `json:"downloadFolder"`
}

func NewDownloader(settingsPath string) (Downloader, error) {
	exists, err := fileExists(settingsPath)
	if err != nil {
		return Downloader{}, err
	}

	downloader := Downloader{}
	if exists {
		contents, err := os.ReadFile(settingsPath)
		if err != nil {
			return Downloader{}, err
		}

		err = json.Unmarshal(contents, &downloader)
		if err != nil {
			return Downloader{}, err
		}
	} else {
		downloader.Transfers = make(map[string]TransferInfo)
		downloader.States = make(map[string]*TransferState)
	}

	// set a default path
	if len(strings.TrimSpace(downloader.DownloadFolder)) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		downloader.DownloadFolder = path.Join(home, "Downloads")
	}

	// open files associated to ongoing transfers
	for transferId, info := range downloader.Transfers {
		state := downloader.States[transferId]
		fullpath := path.Join(downloader.DownloadFolder, info.FileName)
		state.file, err = OpenFile(fullpath, info.FileSize)
		if err != nil {
			return Downloader{}, err
		}
	}

	return downloader, nil
}

func (d *Downloader) Close(settingsFile string) error {
	for _, state := range d.States {
		state.file.Flush()
		state.file.Unmap()
	}

	// save state -- TODO: test later
	//jsonData, err := json.Marshal(*d)
	//if err != nil {
	//	return err
	//}
	//return os.WriteFile(settingsFile, jsonData, 0644)
	return nil
}

func (d *Downloader) HandleMessage(msg Message) Message {
	switch msg.MessageType {
	case TRANSFER_INFO:
		info, err := DeserializeInto[TransferInfo](msg)
		if err != nil {
			panic(err)
		}
		return d.handleInfo(info)
	case TRANSFER_CHUNK:
		chunk, err := DeserializeInto[FileChunk](msg)
		if err != nil {
			panic(err)
		}
		return d.handleChunk(chunk)
	}
	log.Panicf("Received unkonwn message: %v\n", msg)
	return Message{}
}

func (d *Downloader) handleInfo(info TransferInfo) Message {
	fullpath := path.Join(d.DownloadFolder, info.FileName)
	contents, err := OpenFile(fullpath, info.FileSize)
	if err != nil {
		panic(err)
	}

	state := &TransferState{
		TransferId:     info.TransferId,
		AmountReceived: 0,
		file:           contents,
	}

	d.States[info.TransferId] = state
	d.Transfers[info.TransferId] = info

	return NewMessage(TRANSFER_STATE, *state)
}

func (d *Downloader) handleChunk(chunk FileChunk) Message {
	info := d.Transfers[chunk.TransferId]
	state := d.States[chunk.TransferId]
	chunkSize := int64(len(chunk.Data))

	// already got this chunk, ignore
	if state.AmountReceived >= chunk.Offset+chunkSize {
		return NewMessage(TRANSFER_STATE, *state)
	}

	if err := state.file.Lock(); err != nil {
		panic(err)
	}
	copy(state.file[chunk.Offset:], chunk.Data)
	if err := state.file.Unlock(); err != nil {
		panic(err)
	}
	if err := state.file.Flush(); err != nil { // TODO: faster way?
		panic(err)
	}

	state.AmountReceived = chunk.Offset + chunkSize
	if state.AmountReceived >= info.FileSize { // last chunk, done writing
		state.file.Unmap()
	}

	return NewMessage(TRANSFER_STATE, *state)
}
