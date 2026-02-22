package main

import (
	"log"
	"os"

	"github.com/tarunhawdia/myTorrent/bencode"
	"github.com/tarunhawdia/myTorrent/p2p"
	"github.com/tarunhawdia/myTorrent/tracker"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: myTorrent <path/to/torrent_file> <output_file_path>")
	}

	torPath := os.Args[1]
	outPath := os.Args[2]

	tf, err := bencode.ParseTorrentFile(torPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Connecting to tracker: %s", tf.Announce)
	if tf.Announce == "" {
		log.Fatalf("Announce URL is empty in parsed torrent file")
	}

	// Make a tracker.TorrentFile
	ttf := &tracker.TorrentFile{
		Announce: tf.Announce,
		InfoHash: tf.InfoHash,
		Length:   tf.Length,
	}

	peersResp, err := tracker.RequestPeers(ttf)
	if err != nil {
		log.Fatalf("Could not get peers from tracker: %s", err)
	}

	log.Printf("Found %d peers", len(peersResp.Peers))

	// Generate a PeerID (can be derived or just random, we'll use same from tracker or just generate anew if tracker didn't expose it)
	// Actually tracker uses a global peerID but let's just make one here or expose it.
	// Since tracker is in its own package and peerID is private, we should expose it or recreate.
	// Let's just create one:
	var peerID [20]byte
	copy(peerID[:], "-MY0001-123456789012")

	torrent := p2p.Torrent{
		Peers:       peersResp.Peers,
		PeerID:      peerID,
		InfoHash:    tf.InfoHash,
		PieceHashes: tf.Pieces,
		PieceLength: tf.PieceLength,
		Length:      tf.Length,
		Name:        tf.Name,
	}

	buf, err := torrent.Download()
	if err != nil {
		log.Fatal(err)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	_, err = outFile.Write(buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Download complete! Saved to %s", outPath)
}
