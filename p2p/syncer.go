package p2p

import (
	"fmt"
	"log"
	"path"

	"github.com/edsrzf/mmap-go"
)

// file transfer message types
type Transfer struct {
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

type Reply struct {
	ChunksReceived []bool `json:"received"`
}

type FileSyncer struct {
	Transfers      map[string]Transfer  // map TransferId to Transfer
	files          map[string]mmap.MMap // map TransferId to a file
	downloadFolder string
}

func NewFileSyncer(downloadFolder string) FileSyncer {
	// TODO; load Transfers hashmap from a file
	// and open files for all the entries that are there

	return FileSyncer{
		Transfers:      make(map[string]Transfer),
		files:          make(map[string]mmap.MMap),
		downloadFolder: downloadFolder,
	}
}

func (f *FileSyncer) Close() {
	// TODO: save the Transfers hashmap to a file
	for _, file := range f.files {
		file.Flush()
		file.Unmap()
	}
}

func (f *FileSyncer) TransferRecipients(transferId string) ([]string, error) {
	_, exists := f.Transfers[transferId]
	if !exists {
		return nil, fmt.Errorf("unknown file transfer %s", transferId)
	}
	return f.Transfers[transferId].Recipients, nil
}

func (f *FileSyncer) SenderMarkTransfer(info Transfer) {
	f.Transfers[info.TransferId] = info
}

func (f *FileSyncer) HandleMessage(msg Message) *Message {
	switch msg.MessageType {
	case TRANSFER_START:
		info, err := DeserializeInto[Transfer](msg)
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
		reply, err := DeserializeInto[Reply](msg)
		if err != nil {
			panic(err)
		}
		return f.handleReply(reply)
	}
}

func (f *FileSyncer) handleInfo(info Transfer) *Message {
	fullpath := path.Join(f.downloadFolder, info.FileName)
	contents, err := OpenFile(fullpath, info.FileSize)
	if err != nil {
		panic(err)
	}

	f.files[info.TransferId] = contents
	f.Transfers[info.TransferId] = info

	msg := NewMessage(TRANSFER_REPLY, Reply{})
	return &msg
}

func (f *FileSyncer) handleChunk(chunk FileChunk) *Message {
	info := f.Transfers[chunk.TransferId]
	file := f.files[chunk.TransferId]

	file.Lock()
	copy(file[info.ChunkSize*chunk.Index:], chunk.Data)
	file.Unlock()
	file.Flush() // TODO: faster way?

	// last chunk, done writing
	if chunk.Index == info.NumChunks-1 {
		file.Unmap()
		delete(f.files, info.TransferId)
	}

	msg := NewMessage(TRANSFER_REPLY, Reply{})
	return &msg
}

func (f *FileSyncer) handleReply(reply Reply) *Message {
	log.Println("Got a reply!")
	// TODO: last big question: what should this reply be?????
	return nil
}
