package p2p

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/edsrzf/mmap-go"
)

// file transfer message types
type TransferInfo struct {
	TransferId string `json:"transferId"`
	Recipient  string `json:"recipient"`
	FileName   string `json:"name"`
	FileSize   int64  `json:"size"`
}

type TransferChunk struct {
	TransferId string  `json:"transferId"`
	Recipient  string  `json:"recipient"`
	Data       []uint8 `json:"data"`
	Offset     int64   `json:"offset"`
}

type TransferCancel struct {
	TransferId string `json:"transferId"`
	Recipient  string `json:"recipient"`
}

type TransferResponse struct {
	TransferId     string `json:"transferId"`
	AmountReceived int64  `json:"amountReceived"`
	Cancelled      bool   `json:"cancelled"`
	file           mmap.MMap
}

type Downloader struct {
	mutex          sync.Mutex
	Transfers      map[string]TransferInfo      `json:"transfers"`
	States         map[string]*TransferResponse `json:"states"`
	DownloadFolder string                       `json:"downloadFolder"`
}

func NewDownloader(settingsPath string) (*Downloader, error) {
	exists, err := fileExists(settingsPath)
	if err != nil {
		return nil, err
	}

	downloader := &Downloader{}
	if exists {
		contents, err := os.ReadFile(settingsPath)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(contents, &downloader)
		if err != nil {
			return nil, err
		}
	} else {
		downloader.Transfers = make(map[string]TransferInfo)
		downloader.States = make(map[string]*TransferResponse)
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
			return nil, err
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
		chunk, err := DeserializeInto[TransferChunk](msg)
		if err != nil {
			panic(err)
		}
		return d.handleChunk(chunk)
	case TRANSFER_CANCEL:
		info, err := DeserializeInto[TransferCancel](msg)
		if err != nil {
			panic(err)
		}
		return d.handleCancel(info.TransferId)
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

	d.mutex.Lock()
	defer d.mutex.Unlock()

	state := &TransferResponse{
		TransferId:     info.TransferId,
		AmountReceived: 0,
		file:           contents,
	}

	d.States[info.TransferId] = state
	d.Transfers[info.TransferId] = info
	return NewMessage(TRANSFER_RESPONSE, *state)
}

func (d *Downloader) handleChunk(chunk TransferChunk) Message {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	state, exists := d.States[chunk.TransferId]
	if !exists { // must have been deleted
		t := TransferResponse{TransferId: chunk.TransferId, Cancelled: true}
		return NewMessage(TRANSFER_RESPONSE, t)
	}

	info := d.Transfers[chunk.TransferId]
	chunkSize := int64(len(chunk.Data))

	// already got this chunk, ignore
	if state.AmountReceived >= chunk.Offset+chunkSize {
		return NewMessage(TRANSFER_RESPONSE, *state)
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
	return NewMessage(TRANSFER_RESPONSE, *state)
}

func (d *Downloader) handleCancel(transferId string) Message {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if err := d.States[transferId].file.Unmap(); err != nil {
		panic(err)
	}

	fullpath := path.Join(d.DownloadFolder, d.Transfers[transferId].FileName)
	if err := os.Remove(fullpath); err != nil {
		panic(err)
	}

	delete(d.States, transferId)
	delete(d.Transfers, transferId)

	state := TransferResponse{TransferId: transferId, Cancelled: true}
	return NewMessage(TRANSFER_RESPONSE, state)
}
