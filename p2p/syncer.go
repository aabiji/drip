package p2p

import (
	"fmt"
	"path"

	"github.com/edsrzf/mmap-go"
)

// file transfer message types
type TransferInfo struct {
	TransferId string   `json:"transfer_id"`
	Recipients []string `json:"recipients"`
	FileName   string   `json:"name"`
	FileSize   int64    `json:"size"`
	NumChunks  int      `json:"numChunks"`
	ChunkSize  int      `json:"chunkSize"`
}

type FileChunk struct {
	TransferId string  `json:"transfer_id"`
	Data       []uint8 `json:"data"`
	Index      int     `json:"chunkIndex"`
}

type TransferState struct {
	TransferId     string `json:"transfer_id"`
	ChunksReceived []bool `json:"chunks_received"`
	file           mmap.MMap
}

type FileSyncer struct {
	transfers      map[string]TransferInfo
	states         map[string]*TransferState
	downloadFolder string
}

func NewFileSyncer(downloadFolder string) FileSyncer {
	// TODO; load transfers hashmap from a file
	// and open states for all the entries that are there

	return FileSyncer{
		transfers:      make(map[string]TransferInfo),
		states:         make(map[string]*TransferState),
		downloadFolder: downloadFolder,
	}
}

func (f *FileSyncer) Close() {
	// TODO: save the transfers hashmap to a file
	for _, state := range f.states {
		state.file.Flush()
		state.file.Unmap()
	}
}

func (f *FileSyncer) TransferRecipients(transferId string) ([]string, error) {
	_, exists := f.transfers[transferId]
	if !exists {
		return nil, fmt.Errorf("unknown file transfer %s", transferId)
	}
	return f.transfers[transferId].Recipients, nil
}

func (f *FileSyncer) SenderMarkTransfer(info TransferInfo) {
	f.transfers[info.TransferId] = info // TODO: send this to the frontend on app open
	// TODO: the frontend should spawn a web worker to start transferring files
}

func (f *FileSyncer) HandleMessage(msg Message) *Message {
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
	default:
		// do nothing...
		// NOTE: The app's frontend is the one handling peer replies
		// the backend doesn't actually need to keep track of what chunks
		// it has sent since the recipients will tell us that info
		return nil
	}
}

func (f *FileSyncer) handleInfo(info TransferInfo) *Message {
	fullpath := path.Join(f.downloadFolder, info.FileName)
	contents, err := OpenFile(fullpath, info.FileSize)
	if err != nil {
		panic(err)
	}

	state := &TransferState{
		TransferId:     info.TransferId,
		ChunksReceived: make([]bool, info.NumChunks),
		file:           contents,
	}

	f.states[info.TransferId] = state
	f.transfers[info.TransferId] = info

	msg := NewMessage(TRANSFER_STATE, *state)
	return &msg
}

func (f *FileSyncer) handleChunk(chunk FileChunk) *Message {
	info := f.transfers[chunk.TransferId]
	state := f.states[chunk.TransferId]

	if err := state.file.Lock(); err != nil {
		panic(err)
	}
	copy(state.file[info.ChunkSize*chunk.Index:], chunk.Data)
	if err := state.file.Unlock(); err != nil {
		panic(err)
	}
	if err := state.file.Flush(); err != nil { // TODO: faster way?
		panic(err)
	}

	state.ChunksReceived[chunk.Index] = true
	// last chunk, done writing
	if chunk.Index == info.NumChunks-1 {
		state.file.Unmap()
	}
}
