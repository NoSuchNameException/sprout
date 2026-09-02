package transport

import (
	"encoding/binary"
	"fmt"

	"google.golang.org/grpc/encoding"
)

var _ encoding.Codec = (*codec)(nil)

// codec implements [encoding.Codec] to directly frame raw byte slices into Protobuf-compatible
// length-delimited messages without generating intermediate Protobuf structs.
type codec struct{}

// Name returns the unique identifier for the codec, registered with gRPC as "vless-proto".
func (codec) Name() string { return "vless-proto" }

// Marshal encodes a raw []byte slice into a Protobuf wire-format message.
// The resulting binary format consists of:
//   - Field Tag 1 with wire type 2 (0x0a)
//   - Varint-encoded length of data
//   - Raw payload bytes
func (codec) Marshal(v any) ([]byte, error) {
	data, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("codec: expected []byte, got %T", v)
	}

	// Pre-allocate buffer with exact capacity: 1 byte (tag) + max 10 bytes (varint len) + payload len
	pb := make([]byte, 0, 1+binary.MaxVarintLen64+len(data))
	pb = append(pb, 0x0a)
	pb = binary.AppendUvarint(pb, uint64(len(data)))
	pb = append(pb, data...)

	return pb, nil
}

// Unmarshal decodes a Protobuf length-delimited message into a target *[]byte pointer.
// It verifies the field tag (0x0a), checks varint bounds, and copies the payload
// into the existing buffer without allocating new memory where possible.
func (codec) Unmarshal(data []byte, v any) error {
	ptr, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("codec: expected *[]byte, got %T", v)
	}

	if len(data) < 1 {
		return fmt.Errorf("codec: empty buffer")
	}
	if data[0] != 0x0a {
		return fmt.Errorf("codec: unexpected field tag: 0x%x", data[0])
	}

	value, n := binary.Uvarint(data[1:])
	if n <= 0 {
		return fmt.Errorf("codec: invalid varint")
	}

	start := 1 + n
	if value > uint64(len(data)-start) {
		return fmt.Errorf("codec: truncated frame")
	}
	end := start + int(value)

	*ptr = append((*ptr)[:0], data[start:end]...)
	return nil
}
