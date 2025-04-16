package bencode

import (
	"io"
	"strconv"
)

func (d *decoder) parseString() (string, error) {
	lenStr, err := d.readUntil(':')
	if err != nil {
		return "", err
	}
	length, err := strconv.Atoi(string(lenStr))
	if err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}
