package bencode

import (
	"fmt"
	"io"
	"reflect"
	"sort"
)

func Encode(w io.Writer, val interface{}) error {
	switch v := val.(type) {
	case string:
		_, err := fmt.Fprintf(w, "%d:%s", len(v), v)
		return err
	case int:
		_, err := fmt.Fprintf(w, "i%de", v)
		return err
	case int64:
		_, err := fmt.Fprintf(w, "i%de", v)
		return err
	case []interface{}:
		_, err := io.WriteString(w, "l")
		if err != nil {
			return err
		}
		for _, item := range v {
			if err := Encode(w, item); err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, "e")
		return err
	case map[string]interface{}:
		_, err := io.WriteString(w, "d")
		if err != nil {
			return err
		}
		// Sort keys (required by bencode spec)
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := Encode(w, k); err != nil {
				return err
			}
			if err := Encode(w, v[k]); err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, "e")
		return err
	default:
		return fmt.Errorf("unsupported type: %s", reflect.TypeOf(val))
	}
}
