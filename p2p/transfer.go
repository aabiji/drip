package p2p

type TransferInfo struct {
	Recipients []string `json:"recipients"`
	FileName   string   `json:"name"`
	FileSize   int64    `json:"size"`
	NumChunks  int      `json:"numChunks"`
}

type FileChunk struct {
	Data  []uint8 `json:"data"`
	Index int     `json:"chunkIndex"`
}
