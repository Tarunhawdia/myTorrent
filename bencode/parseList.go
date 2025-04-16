package bencode

func (d *decoder) parseList() ([]interface{}, error) {
	var list []interface{}
	for {
		c, err := d.r.Peek(1)
		if err != nil {
			return nil, err
		}
		if c[0] == 'e' {
			d.r.ReadByte()
			break
		}
		val, err := d.parse()
		if err != nil {
			return nil, err
		}
		list = append(list, val)
	}
	return list, nil
}
