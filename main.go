package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/tarunhawdia/myTorrent/bencode"
)

func main() {
	file, err := os.Open("alice.torrent")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	result, err := bencode.Decode(bufio.NewReader(file))
	if err != nil {
		panic(err)
	}

	torrent, ok := result.(map[string]interface{})
	if !ok {
		panic("Expected top-level dict")
	}

	// Print info dictionary
	if info, ok := torrent["info"]; ok {
		fmt.Printf("Info dictionary: %#v\n", info)
	}
}
