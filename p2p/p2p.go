package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/tarunhawdia/myTorrent/client"
	"github.com/tarunhawdia/myTorrent/message"
	"github.com/tarunhawdia/myTorrent/tracker"
)

// MaxBlockSize is the largest number of bytes a request can ask for
const MaxBlockSize = 16384 // 16 KB

// MaxBacklog is the number of unfulfilled requests a client can have in its pipeline
const MaxBacklog = 5

// pieceWork represents a piece that needs to be downloaded.
type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

// pieceResult represents a successfully downloaded piece.
type pieceResult struct {
	index int
	buf   []byte
}

// pieceProgress keeps track of the downloaded blocks for a piece.
type pieceProgress struct {
	index      int
	client     *client.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
}

// readMessage reads a message from the client and updates state.
func (state *pieceProgress) readMessage() error {
	msg, err := state.client.Read() // this call blocks
	if err != nil {
		return err
	}

	if msg == nil { // keep-alive
		return nil
	}

	switch msg.ID {
	case message.MsgChoke:
		state.client.Choked = true
	case message.MsgUnchoke:
		state.client.Choked = false
	case message.MsgHave:
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		// update Bitfield? We handle this simply by ignoring and relying on initial bitfield for now,
		// or we could update it. Not strictly necessary for a basic client.
		_ = index
	case message.MsgPiece:
		n, err := message.ParsePiece(state.index, state.buf, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}
	return nil
}

func attemptDownloadPiece(c *client.Client, pw *pieceWork) ([]byte, error) {
	state := pieceProgress{
		index:  pw.index,
		client: c,
		buf:    make([]byte, pw.length),
	}

	// Setting a deadline helps get unresponsive peers unstuck
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{})

	for state.downloaded < pw.length {
		if !state.client.Choked {
			for state.backlog < MaxBacklog && state.requested < pw.length {
				blockSize := MaxBlockSize
				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				err := c.SendRequest(pw.index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return state.buf, nil
}

func checkIntegrity(pw *pieceWork, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], pw.hash[:]) {
		return fmt.Errorf("index %d failed integrity check", pw.index)
	}
	return nil
}

// Torrent holds data required to download a torrent from a list of peers.
type Torrent struct {
	Peers       []tracker.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

// calculateBounds gets the start/end offsets for a piece.
func (t *Torrent) calculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}

// calculatePieceSize computes the size of a piece.
func (t *Torrent) calculatePieceSize(index int) int {
	begin, end := t.calculateBoundsForPiece(index)
	return end - begin
}

// peerWorker connects to a peer and pulls pieces from the work queue.
func (t *Torrent) peerWorker(peer tracker.Peer, workQueue chan *pieceWork, results chan *pieceResult) {
	c, err := client.New(peer, t.PeerID, t.InfoHash)
	if err != nil {
		log.Printf("Could not handshake with %s. Disconnecting.", peer.IP)
		return
	}
	defer c.Conn.Close()

	c.SendUnchoke()
	c.SendInterested()

	for pw := range workQueue {
		if !c.HasPiece(pw.index) {
			workQueue <- pw // Put piece back on the queue
			continue
		}

		buf, err := attemptDownloadPiece(c, pw)
		if err != nil {
			log.Printf("Exiting: %s", err)
			workQueue <- pw // Put piece back on the queue
			return
		}

		err = checkIntegrity(pw, buf)
		if err != nil {
			log.Printf("Piece #%d failed integrity check", pw.index)
			workQueue <- pw // Put piece back on the queue
			continue
		}

		c.SendHave(pw.index)
		results <- &pieceResult{pw.index, buf}
	}
}

// Download downloads the torrent and returns the full file buffer.
func (t *Torrent) Download() ([]byte, error) {
	log.Printf("Starting download for %s", t.Name)

	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)
	for index, hash := range t.PieceHashes {
		length := t.calculatePieceSize(index)
		workQueue <- &pieceWork{index, hash, length}
	}

	// Start workers
	var wg sync.WaitGroup
	for _, peer := range t.Peers {
		wg.Add(1)
		go func(p tracker.Peer) {
			defer wg.Done()
			t.peerWorker(p, workQueue, results)
		}(peer)
	}

	// Goroutine to close results when all workers exit
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res, ok := <-results
		if !ok {
			return nil, fmt.Errorf("all peer workers exited without completing the download (done %d/%d)", donePieces, len(t.PieceHashes))
		}
		begin, end := t.calculateBoundsForPiece(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++

		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		numWorkers := runtime.NumGoroutine() - 2 // subtract main thread & closer thread
		log.Printf("(%0.2f%%) Downloaded piece #%d from %d peers", percent, res.index, numWorkers)
	}
	close(workQueue) // Safe to close after we received all pieces
	return buf, nil
}
