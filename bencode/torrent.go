package bencode

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"errors"
	"os"
)

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceLength int
	Pieces      [][20]byte
	Name        string
	Length      int
}

func ParseTorrentFile(path string) (*TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := Decode(bufio.NewReader(file))
	if err != nil {
		return nil, err
	}

	torrentMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid torrent file format")
	}

	// Extract announce
	announce, _ := torrentMap["announce"].(string)

	// Extract and re-encode info
	infoMap, ok := torrentMap["info"].(map[string]interface{})
	if !ok {
		return nil, errors.New("info dictionary missing")
	}

	var buf bytes.Buffer
	err = Encode(&buf, infoMap)
	if err != nil {
		return nil, err
	}
	infoHash := sha1.Sum(buf.Bytes())

	// Extract individual fields from infoMap
	name, _ := infoMap["name"].(string)
	length, _ := infoMap["length"].(int)
	if length == 0 {
		length64, _ := infoMap["length"].(int64)
		length = int(length64)
	}
	pieceLength, _ := infoMap["piece length"].(int)
	if pieceLength == 0 {
		pl64, _ := infoMap["piece length"].(int64)
		pieceLength = int(pl64)
	}

	piecesRaw, ok := infoMap["pieces"].(string)
	if !ok {
		return nil, errors.New("pieces field is not a string")
	}
	pieces := parsePieces([]byte(piecesRaw))

	return &TorrentFile{
		Announce:    announce,
		InfoHash:    infoHash,
		PieceLength: pieceLength,
		Pieces:      pieces,
		Name:        name,
		Length:      length,
	}, nil
}

func parsePieces(b []byte) [][20]byte {
	numHashes := len(b) / 20
	var pieces [][20]byte
	for i := 0; i < numHashes; i++ {
		var hash [20]byte
		copy(hash[:], b[i*20:(i+1)*20])
		pieces = append(pieces, hash)
	}
	return pieces
}
