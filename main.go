package main

import (
	"fmt"
	"log"

	"github.com/tarunhawdia/myTorrent/bencode"
)

func main() {
	tf, err := bencode.ParseTorrentFile("alice.torrent")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Announce URL:", tf.Announce)
	fmt.Printf("Info Hash: %x\n", tf.InfoHash)
	fmt.Println("Piece Length:", tf.PieceLength)
	fmt.Println("File Name:", tf.Name)
	fmt.Println("File Length:", tf.Length)
	fmt.Println("Number of Pieces:", len(tf.Pieces))
}
