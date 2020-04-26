package cbcio

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"testing"
)

type compareWriter struct {
	rd io.Reader
}

func (w *compareWriter) Write(p []byte) (n int, err error) {
	buf := make([]byte, len(p))
	n, err = io.ReadFull(w.rd, buf)
	if err != nil {
		return
	}

	if !bytes.Equal(buf, p) {
		return n, errors.New("difference found")
	}

	return
}

func TestWriter(t *testing.T) {
	decFile, err := os.Open("testdata/media_b1973000_1_dec.bigtest")
	if err != nil {
		t.Fatal(err)
	}
	defer decFile.Close()

	cw := &compareWriter{rd: decFile}

	key, _ := hex.DecodeString("5E0ECC501DBC368679689947B5E8D5E1")
	iv, _ := hex.DecodeString("00000000000000000000000000000001")

	testFile, err := os.Open("testdata/media_b1973000_1.bigtest")
	if err != nil {
		t.Fatal(err)
	}
	defer decFile.Close()

	bufSizes := []int{1, 7, 15, 32 * 1024}
	for _, bufSize := range bufSizes {
		_, _ = testFile.Seek(0, 0)
		_, _ = decFile.Seek(0, 0)

		w := NewWriter(cw, key, iv)
		_, err = io.CopyBuffer(w, testFile, make([]byte, bufSize))
		if err != nil {
			t.Errorf("%v when buf size is %d", err, bufSize)
		}

		err = w.Final()
		if err != nil {
			t.Errorf("%v when buf size is %d", err, bufSize)
		}
	}
}
