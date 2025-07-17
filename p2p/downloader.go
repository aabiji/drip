package p2p

import (
	"log"
	"path"

	"github.com/edsrzf/mmap-go"
)

// file transfer message types
type TransferInfo struct {
	TransferId string   `json:"transferId"`
	Recipients []string `json:"recipients"`
	FileName   string   `json:"name"`
	FileSize   int64    `json:"size"`
}

type FileChunk struct {
	TransferId string   `json:"transferId"`
	Recipients []string `json:"recipients"`
	Data       []uint8  `json:"data"`
}

type TransferState struct {
	TransferId     string `json:"transferId"`
	AmountReceived int64  `json:"amountReceived"`
	file           mmap.MMap
}

type Downloader struct {
	transfers      map[string]TransferInfo
	states         map[string]*TransferState
	downloadFolder string
}

func NewDownloader(downloadFolder string) Downloader {
	// TODO; load transfers hashmap from a file
	// and open states for all the entries that are there

	return Downloader{
		transfers:      make(map[string]TransferInfo),
		states:         make(map[string]*TransferState),
		downloadFolder: downloadFolder,
	}
}

func (f *Downloader) Close() {
	// TODO: save the transfers hashmap to a file
	for _, state := range f.states {
		state.file.Flush()
		state.file.Unmap()
	}
}

func (f *Downloader) HandleMessage(msg Message) Message {
	switch msg.MessageType {
	case TRANSFER_INFO:
		info, err := DeserializeInto[TransferInfo](msg)
		if err != nil {
			panic(err)
		}
		return f.handleInfo(info)
	case TRANSFER_CHUNK:
		chunk, err := DeserializeInto[FileChunk](msg)
		if err != nil {
			panic(err)
		}
		return f.handleChunk(chunk)
	}
	log.Panicf("Received unkonwn message: %v\n", msg)
	return Message{}
}

func (f *Downloader) handleInfo(info TransferInfo) Message {
	fullpath := path.Join(f.downloadFolder, info.FileName)
	contents, err := OpenFile(fullpath, info.FileSize)
	if err != nil {
		panic(err)
	}

	state := &TransferState{
		TransferId:     info.TransferId,
		AmountReceived: 0,
		file:           contents,
	}

	f.states[info.TransferId] = state
	f.transfers[info.TransferId] = info

	return NewMessage(TRANSFER_STATE, *state)
}

func (f *Downloader) handleChunk(chunk FileChunk) Message {
	info := f.transfers[chunk.TransferId]
	state := f.states[chunk.TransferId]

	if err := state.file.Lock(); err != nil {
		panic(err)
	}
	copy(state.file[state.AmountReceived:], chunk.Data)
	if err := state.file.Unlock(); err != nil {
		panic(err)
	}
	if err := state.file.Flush(); err != nil { // TODO: faster way?
		panic(err)
	}

	state.AmountReceived += int64(len(chunk.Data))
	if state.AmountReceived >= info.FileSize { // last chunk, done writing
		state.file.Unmap()
	}

	return NewMessage(TRANSFER_STATE, *state)
}
