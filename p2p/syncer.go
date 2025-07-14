package p2p

// file transfer message types
type FileInfo struct {
	Recipients []string `json:"recipients"`
	FileName   string   `json:"name"`
	FileSize   int64    `json:"size"`
	NumChunks  int      `json:"numChunks"`
}

type FileChunk struct {
	Data       []uint8  `json:"data"`
	Index      int      `json:"chunkIndex"`
	Recipients []string `json:"recipients"`
}

type Reply struct {
	ChunksReceived []bool `json:"received"`
}

// file transfer coordinator
type FileSyncer struct{}

func NewFileSyncer() FileSyncer {
	return FileSyncer{}
}

func (f *FileSyncer) HandleMessage(msg Message) Message {
	switch msg.MessageType {
	case TRANSFER_START:
		info, err := DeserializeInto[FileInfo](msg)
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

// TODO!
func (f *FileSyncer) handleInfo(info FileInfo) Message {
	return NewMessage(TRANSFER_REPLY, Reply{})
}

func (f *FileSyncer) handleChunk(chunk FileChunk) Message {
	return NewMessage(TRANSFER_REPLY, Reply{})
}

func (f *FileSyncer) handleReply(reply Reply) Message {
	return NewMessage(TRANSFER_REPLY, Reply{})
}
