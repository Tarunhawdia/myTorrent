package tracker

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/tarunhawdia/myTorrent/bencode"
)

// Peer represents a single peer discovered by the tracker.
type Peer struct {
	IP   net.IP
	Port uint16
}

// TrackerResponse represents the bencoded response from the tracker.
// We expect a dictionary containing 'peers' (a byte string in this case).
type TrackerResponse struct {
	Interval int
	Peers    []Peer // List of peers, decoded from the byte string
}

// TorrentFile is a minimal struct mirroring bencode.TorrentFile
// only containing fields necessary for the tracker request.
type TorrentFile struct {
	Announce string
	InfoHash [20]byte
	Length   int
}

// peerID is the unique identifier for our client.
var peerID string

func init() {
	// Generate a unique 20-byte Peer ID (used to identify our client to the network)
	// Common format: -IDXXXYYYY-ZZZZZZZZZZZZ (20 bytes)
	// Example: -TR2940-a0m48s17d7b0
	prefix := "-MY0001-" // 8 bytes identifying the client
	suffix := time.Now().Format("20060102150405")

	if len(prefix)+len(suffix) > 20 {
		peerID = prefix + suffix[:20-len(prefix)]
	} else {
		peerID = prefix + suffix
	}
}

// decodePeers takes the bencoded byte string of peers and converts it into a slice of Peer structs.
// The peer string is a concatenation of (IP address, Port) pairs.
// IP is 4 bytes, Port is 2 bytes (Big Endian). Total 6 bytes per peer.
func decodePeers(peersBin string) ([]Peer, error) {
	const peerSize = 6 // 4 (IP) + 2 (Port)
	buf := []byte(peersBin)

	if len(buf)%peerSize != 0 {
		return nil, fmt.Errorf("tracker: received malformed peers list. Length %d is not a multiple of %d", len(buf), peerSize)
	}

	numPeers := len(buf) / peerSize
	p := make([]Peer, 0, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * peerSize

		ip := net.IP(buf[offset : offset+4])
		// Port is 2 bytes in Big Endian format
		port := uint16(buf[offset+4])<<8 | uint16(buf[offset+5])

		p = append(p, Peer{IP: ip, Port: port})
	}
	return p, nil
}

// buildTrackerURL constructs the URL for the tracker request.
func buildTrackerURL(tf *TorrentFile) (string, error) {
	u, err := url.Parse(tf.Announce)
	if err != nil {
		return "", fmt.Errorf("tracker: failed to parse announce URL: %w", err)
	}

	params := url.Values{}
	params.Set("info_hash", string(tf.InfoHash[:])) // URL-encoded InfoHash
	params.Set("peer_id", peerID)                   // Our unique client ID
	params.Set("port", "6881")                      // Standard client port
	params.Set("uploaded", "0")                     // Initial uploaded bytes
	params.Set("downloaded", "0")                   // Initial downloaded bytes
	params.Set("left", strconv.Itoa(tf.Length))     // Total bytes left to download
	params.Set("compact", "1")                      // Request compact peer list format
	params.Set("event", "started")                  // Inform tracker we are starting
	params.Set("numwant", "50")                     // Request 50 peers instead of default

	u.RawQuery = params.Encode()
	return u.String(), nil
}

// RequestPeers sends an HTTP GET request to the tracker and returns the list of Peers.
func RequestPeers(tf *TorrentFile) (*TrackerResponse, error) {
	url, err := buildTrackerURL(tf)
	if err != nil {
		return nil, err
	}
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("tracker: failed to connect to tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for potential error message from tracker
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tracker: received non-200 status code: %d. Body: %s", resp.StatusCode, string(body))
	}

	// Read and Decode the Bencoded response
	decoded, err := bencode.Decode(bufio.NewReader(resp.Body))
	if err != nil {
		return nil, fmt.Errorf("tracker: failed to decode Bencode response: %w", err)
	}

	responseDict, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, errors.New("tracker: Bencode response is not a dictionary")
	}

	// 1. Extract Interval
	interval := 0
	if i, ok := responseDict["interval"].(int64); ok {
		interval = int(i)
	}

	// 2. Extract and Decode Peers (Peers are expected as a byte string in compact format)
	peersBin, ok := responseDict["peers"].(string)
	if !ok {
		// This happens if the tracker is not using the compact format (which we requested)
		// or if the key is missing. For now, we assume compact format is returned.
		return nil, errors.New("tracker: 'peers' key missing or not in expected string format (compact)")
	}

	peers, err := decodePeers(peersBin)
	if err != nil {
		return nil, fmt.Errorf("tracker: failed to decode peer list: %w", err)
	}

	// 3. Assemble final response
	tr := TrackerResponse{
		Interval: interval,
		Peers:    peers,
	}

	return &tr, nil
}
