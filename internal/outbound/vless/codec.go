package vless

import (
	"fmt"

	"google.golang.org/grpc/encoding"
)

var _ encoding.Codec = (*vlessCodec)(nil)

type vlessCodec struct{}

func (vlessCodec) Name() string { return "vless-proto" }

func (vlessCodec) Marshal(v any) ([]byte, error) {
	data, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("vlessCodec: expected []byte, got %T", v)
	}

	var pb []byte
	pb = append(pb, 0x0a)
	pb = appendVarint(pb, uint64(len(data)))
	pb = append(pb, data...)

	return pb, nil
}

func (vlessCodec) Unmarshal(data []byte, v any) error {
	ptr, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("vlessCodec: expected *[]byte, got %T", v)
	}

	if len(data) < 1 {
		return fmt.Errorf("vlessCodec: empty buffer")
	}
	if data[0] != 0x0a {
		return fmt.Errorf("vlessCodec: unexpected field tag: 0x%x", data[0])
	}

	value, n := readVarint(data[1:])
	if n == 0 {
		return fmt.Errorf("vlessCodec: invalid varint")
	}

	start := 1 + n
	end := start + int(value)

	if len(data) < end {
		return fmt.Errorf("vlessCodec: truncated frame")
	}

	*ptr = append((*ptr)[:0], data[start:end]...)
	return nil
}

func appendVarint(buf []byte, n uint64) []byte {
	for n >= 0x80 {
		buf = append(buf, byte(n)|0x80)
		n >>= 7
	}
	return append(buf, byte(n))
}

func readVarint(buf []byte) (value uint64, bytesRead int) {
	for i, b := range buf {
		value |= uint64(b&0x7F) << (7 * uint(i))
		if b < 0x80 {
			return value, i + 1
		}
	}
	return 0, 0
}
