package authflow

import (
	"encoding/binary"
	"fmt"
)

// cborDecode decodes one CBOR item from data.
// Returns (decoded, bytesConsumed, error).
// Supports: uint(0), negint(1), bytes(2), text(3), array(4), map(5), tag(6 — passes through).
func cborDecode(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, fmt.Errorf("cbor: empty input")
	}
	major := data[0] >> 5
	info := data[0] & 0x1f

	var n uint64
	head := 1
	switch {
	case info <= 23:
		n = uint64(info)
	case info == 24:
		if len(data) < 2 {
			return nil, 0, fmt.Errorf("cbor: truncated")
		}
		n, head = uint64(data[1]), 2
	case info == 25:
		if len(data) < 3 {
			return nil, 0, fmt.Errorf("cbor: truncated")
		}
		n, head = uint64(binary.BigEndian.Uint16(data[1:3])), 3
	case info == 26:
		if len(data) < 5 {
			return nil, 0, fmt.Errorf("cbor: truncated")
		}
		n, head = uint64(binary.BigEndian.Uint32(data[1:5])), 5
	case info == 27:
		if len(data) < 9 {
			return nil, 0, fmt.Errorf("cbor: truncated")
		}
		n, head = binary.BigEndian.Uint64(data[1:9]), 9
	default:
		return nil, 0, fmt.Errorf("cbor: unsupported additional info %d", info)
	}

	switch major {
	case 0:
		return n, head, nil
	case 1:
		return -int64(n) - 1, head, nil
	case 2:
		end := head + int(n)
		if len(data) < end {
			return nil, 0, fmt.Errorf("cbor: short bytes")
		}
		b := make([]byte, n)
		copy(b, data[head:end])
		return b, end, nil
	case 3:
		end := head + int(n)
		if len(data) < end {
			return nil, 0, fmt.Errorf("cbor: short text")
		}
		return string(data[head:end]), end, nil
	case 4:
		arr := make([]interface{}, n)
		off := head
		for i := range arr {
			v, sz, err := cborDecode(data[off:])
			if err != nil {
				return nil, 0, err
			}
			arr[i], off = v, off+sz
		}
		return arr, off, nil
	case 5:
		m := make(map[interface{}]interface{}, n)
		off := head
		for i := uint64(0); i < n; i++ {
			k, ksz, err := cborDecode(data[off:])
			if err != nil {
				return nil, 0, err
			}
			off += ksz
			v, vsz, err := cborDecode(data[off:])
			if err != nil {
				return nil, 0, err
			}
			m[k], off = v, off+vsz
		}
		return m, off, nil
	case 6: // tag — skip tag value, decode inner item
		v, sz, err := cborDecode(data[head:])
		if err != nil {
			return nil, 0, err
		}
		return v, head + sz, nil
	}
	return nil, 0, fmt.Errorf("cbor: unsupported major type %d", major)
}
