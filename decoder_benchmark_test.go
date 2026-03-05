package yenc

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func BenchmarkDecodeNyuuFixture(b *testing.B) {
	data, err := os.ReadFile("fixture/260731a73db67e8095a5eaf0b64b9d3db0117cdb@nyuu.ntx")
	if err != nil {
		b.Fatalf("read fixture: %v", err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		d, err := Decode(bytes.NewReader(data), DecodeWithBufferSize(200))
		if err != nil {
			b.Fatalf("decode init: %v", err)
		}

		if _, err = io.Copy(io.Discard, d); err != nil {
			b.Fatalf("decode copy: %v", err)
		}
	}
}
