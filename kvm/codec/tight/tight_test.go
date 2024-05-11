package tight

import (
	"slices"
	"testing"
)

func TestEncodeLength(t *testing.T) {
	var bs []byte

	bs = encodeLength(119)
	if !slices.Equal(bs, []byte{119}) {
		t.Fatalf("Expected [119], got %v", bs)
	}

	bs = encodeLength(2434)
	if !slices.Equal(bs, []byte{130, 19}) {
		t.Fatalf("Expected [130, 19], got %v", bs)
	}

	bs = encodeLength(26417)
	if !slices.Equal(bs, []byte{177, 206, 1}) {
		t.Fatalf("Expected [177, 206, 1], got %v", bs)
	}
}

func TestDecodeLength(t *testing.T) {
	var three []byte
	var length, consumedLength int

	three = []byte{119, 0x00, 0x00}
	length, consumedLength = decodeLength(three)
	if length != 119 || consumedLength != 1 {
		t.Fatalf("Expected (119, 1), got (%d, %d)", length, consumedLength)
	}

	three = []byte{130, 19, 0x00}
	length, consumedLength = decodeLength(three)
	if length != 2434 || consumedLength != 2 {
		t.Fatalf("Expected (2434, 2), got (%d, %d)", length, consumedLength)
	}

	three = []byte{164, 23, 0x00}
	length, consumedLength = decodeLength(three)
	if length != 2980 || consumedLength != 2 {
		t.Fatalf("Expected (2980, 2), got (%d, %d)", length, consumedLength)
	}

	three = []byte{213, 1, 0x00}
	length, consumedLength = decodeLength(three)
	if length != 213 || consumedLength != 2 {
		t.Fatalf("Expected (213, 2), got (%d, %d)", length, consumedLength)
	}

	three = []byte{206, 102, 0x00}
	length, consumedLength = decodeLength(three)
	if length != 13134 || consumedLength != 2 {
		t.Fatalf("Expected (13134, 2), got (%d, %d)", length, consumedLength)
	}

	three = []byte{177, 206, 1}
	length, consumedLength = decodeLength(three)
	if length != 26417 || consumedLength != 3 {
		t.Fatalf("Expected (26417, 3), got (%d, %d)", length, consumedLength)
	}
}
