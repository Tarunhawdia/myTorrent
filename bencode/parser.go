package bencode

import (
	"bufio"
	"bytes"
	"fmt"
)

type decoder struct {
	r *bufio.Reader
}

func Decode(r *bufio.Reader) (interface{}, error) {
	d := &decoder{r: bufio.NewReader(r)}
	return d.parse()
}

func (d *decoder) parse() (interface{}, error) {
	c, err := d.r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch {
	case c == 'i':
		return d.parseInt()
	case c == 'l':
		return d.parseList()
	case c == 'd':
		return d.parseDict()
	case c >= '0' && c <= '9':
		d.r.UnreadByte()
		return d.parseString()
	default:
		return nil, fmt.Errorf("Invalid character")
	}

}

func (d *decoder) readUntil(delim byte) ([]byte, error) {
	var buf bytes.Buffer
	for {
		b, err := d.r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == delim {
			break
		}

		buf.WriteByte(b)
	}

	return buf.Bytes(), nil
}