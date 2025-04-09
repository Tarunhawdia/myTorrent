package bencode

import (
	"strconv"
)

func (d *decoder) parseInt() (int64, error) {
	data, err := d.readUntil('e')
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(string(data), 10, 64)
}
