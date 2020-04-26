package cbcio

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"io"
	"m3u8dl/utils"
)

var ErrBadPadding = errors.New("bad padding")

type Writer struct {
	wr  io.Writer
	buf []byte

	decryptBuf []byte

	blockMode cipher.BlockMode
	bSize     int
}

// Write decrypts the input and write it to the underlying writer
// with last block buffered for future un-padding
func (w *Writer) Write(p []byte) (n int, err error) {
	decryptLen := utils.Floor0(len(w.buf)+len(p), w.bSize)

	if decryptLen > 0 {
		// lazy make space for buf
		if w.decryptBuf == nil || len(w.decryptBuf) < decryptLen {
			w.decryptBuf = make([]byte, decryptLen)
		}

		// decrypt the buf and the heading part copied from p
		n += copy(w.buf[len(w.buf):w.bSize], p)
		w.blockMode.CryptBlocks(w.decryptBuf, w.buf[:w.bSize])
		// then decrypt the remaining part
		w.blockMode.CryptBlocks(w.decryptBuf[w.bSize:], p[n:decryptLen-len(w.buf)])

		w.buf = w.buf[:0]
		n += decryptLen - w.bSize

		_, err = w.wr.Write(w.decryptBuf[:decryptLen])
		if err != nil {
			return
		}
	}

	nn := copy(w.buf[len(w.buf):w.bSize], p[n:])
	w.buf = w.buf[:len(w.buf)+nn]
	n += nn

	return
}

// Final will un-pad the last block and write everything to the underlying writer
func (w *Writer) Final() (err error) {
	if len(w.buf) != w.bSize {
		return errors.New("not a whole block buf when calling Flush")
	}

	w.blockMode.CryptBlocks(w.buf, w.buf)

	padLen := int(w.buf[w.bSize-1])
	if padLen == 0 || padLen > w.bSize {
		return ErrBadPadding
	}
	for i := 2; i <= padLen; i++ {
		if int(w.buf[w.bSize-i]) != padLen {
			return ErrBadPadding
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
