package client

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/tarunhawdia/myTorrent/message"
	"github.com/tarunhawdia/myTorrent/tracker"
)

// Client is a TCP connection with a protocol peer.
type Client struct {
	Conn     net.Conn
	Choked   bool
	Bitfield []byte
	peer     tracker.Peer
	infoHash [20]byte
	peerID   [20]byte
}

// CompleteHandshake sends our handshake and reads the peer's handshake.
func CompleteHandshake(conn net.Conn, infoHash, peerID [20]byte) (*message.Handshake, error) {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable deadline

	req := message.NewHandshake(infoHash, peerID)
	_, err := conn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := message.ReadHandshake(conn)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", infoHash, res.InfoHash)
	}

	return res, nil
}

// RecvBitfield parses the initial Bitfield message.
func RecvBitfield(conn net.Conn) ([]byte, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	msg, err := message.Read(conn)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, fmt.Errorf("expected bitfield, got keep-alive")
	}
	if msg.ID != message.MsgBitfield {
		return nil, fmt.Errorf("expected bitfield (ID %d), got ID %d", message.MsgBitfield, msg.ID)
	}

	return msg.Payload, nil
}

// New connects with a peer, completes a handshake, and receives a bitfield.
func New(peer tracker.Peer, peerID, infoHash [20]byte) (*Client, error) {
	peerAddr := net.JoinHostPort(peer.IP.String(), fmt.Sprintf("%d", peer.Port))
	conn, err := net.DialTimeout("tcp", peerAddr, 3*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = CompleteHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bitfield, err := RecvBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		Conn:     conn,
		Choked:   true,
		Bitfield: bitfield,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}

// Read reads a message from the client.
func (c *Client) Read() (*message.Message, error) {
	msg, err := message.Read(c.Conn)
	return msg, err
}

// SendRequest sends a REQUEST message to the peer.
func (c *Client) SendRequest(index, begin, length int) error {
	req := message.FormatRequest(index, begin, length)
	_, err := c.Conn.Write(req.Serialize())
	return err
}

// SendInterested sends an INTERESTED message to the peer.
func (c *Client) SendInterested() error {
	msg := message.Message{ID: message.MsgInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendNotInterested sends a NOT INTERESTED message to the peer.
func (c *Client) SendNotInterested() error {
	msg := message.Message{ID: message.MsgNotInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendUnchoke sends an UNCHOKE message.
func (c *Client) SendUnchoke() error {
	msg := message.Message{ID: message.MsgUnchoke}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendHave sends a HAVE message.
func (c *Client) SendHave(index int) error {
	msg := message.FormatHave(index)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// HasPiece checks if the bitfield indicates the peer has the piece.
func (c *Client) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(c.Bitfield) {
		return false
	}
	return c.Bitfield[byteIndex]>>(7-offset)&1 != 0
}
