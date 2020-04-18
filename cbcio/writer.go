package cbcio

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"io"
	"m3u8dl/utils"
)

type Writer struct {
	wr  io.Writer
	buf []byte

	blockMode cipher.BlockMode
	bSize     int
}

// Write decrypts the input and write it to the underlying writer
// with last block buffered for future un-padding
func (w *Writer) Write(p []byte) (n int, err error) {
	decryptLen := utils.Floor0(len(w.buf)+len(p), w.bSize)

	if decryptLen > 0 {
		decryptBuf := make([]byte, decryptLen)
		// decrypt the buf and the heading part copy from p
		n += copy(w.buf[len(w.buf):w.bSize], p)
		w.blockMode.CryptBlocks(decryptBuf, w.buf[:w.bSize])
		// then decrypt the remaining part
		w.blockMode.CryptBlocks(decryptBuf[w.bSize:], p[n:decryptLen-len(w.buf)])
		w.buf = w.buf[:0]
		n += decryptLen - w.bSize

		_, err = w.wr.Write(decryptBuf)
		if err != nil {
			return
		}
	}

	nn := copy(w.buf[len(w.buf):w.bSize], p[n:])
	w.buf = w.buf[:len(w.buf)+nn]
	n += nn

	return
}

// Flush will un-pad the last block and write everything to the underlying writer
func (w *Writer) Flush() (err error) {
	if len(w.buf) != w.bSize {
		return errors.New("Not a whole block buf when calling Flush")
	}

	w.blockMode.CryptBlocks(w.buf, w.buf)

	padLen := int(w.buf[w.bSize-1])
	errBadPadding := errors.New("bad padding")
	if padLen == 0 || padLen > 16 {
		return errBadPadding
	}
	for i := 2; i <= padLen; i++ {
		if int(w.buf[w.bSize-i]) != padLen {
			return errBadPadding
		}
	}

	_, err = w.wr.Write(w.buf[:w.bSize-padLen])
	return
}

func NewWriter(wr io.Writer, key []byte, iv []byte) *Writer {
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	blockMode := cipher.NewCBCDecrypter(cipherBlock, iv)
	return &Writer{
		wr:        wr,
		buf:       make([]byte, 0, blockMode.BlockSize()),
		blockMode: blockMode,
		bSize:     blockMode.BlockSize(),
	}
}
