package main

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

func TestBitsString2Bytes(t *testing.T) {
	for range 100_000 + rand.Intn(100_000) {
		arrLen := rand.Intn(100)
		if arrLen == 0 {
			arrLen = 1
		}

		var oldBytes []byte
		bitsStr := ""
		for range arrLen {
			randByte := byte(rand.Intn(256))
			bitsStr += fmt.Sprintf("%08b", randByte)
			oldBytes = append(oldBytes, randByte)
		}

		bs, err := BitsString2Bytes(bitsStr)
		if err != nil {
			t.Fatal(err, bitsStr)
		}
		if !slices.Equal(bs, oldBytes) {
			t.Fatalf("expected %v, got %v", oldBytes, bs)
		}
	}
}
