package p2p

// file transfer coordinator
type FileSyncer struct {
	// pending messages to be sent through the peer's data channel
	PendingMessages map[string]chan Message
}

func NewFileSyncer() FileSyncer {
	return FileSyncer{
		PendingMessages: make(map[string]chan Message),
	}
}

func (f *FileSyncer) AddPeer(id string) {
	f.PendingMessages[id] = make(chan Message, 100)
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
