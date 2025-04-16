package bencode

func (d *decoder) parseDict() (map[string]interface{}, error) {

	dict := make(map[string]interface{})
	for {
		c, err := d.r.Peek(1)
		if err != nil {
			return nil, err
		}
		if c[0] == 'e' {
			d.r.ReadByte()
			break
		}
		keyVal, err := d.parseString()
		if err != nil {
			return nil, err
		}
		val, err := d.parse()
		if err != nil {
			return nil, err
		}
		dict[keyVal] = val
	}
	return dict, nil
}
