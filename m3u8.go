package main

import (
	"errors"
	"github.com/grafov/m3u8"
	"io"
	"m3u8dl/utils"
	"os"
)

func decode(input string) (*m3u8.MediaPlaylist, error) {
	var m3u8In io.ReadCloser
	if utils.IsValidUrl(input) {
		resp, err := httpGet(input)
		if err != nil {
			return nil, err
		}
		m3u8In = resp.Body
	} else {
		var err error
		m3u8In, err = os.Open(input)
		if err != nil {
			return nil, err
		}
	}

	pl, listType, err := m3u8.DecodeFrom(m3u8In, true)
	m3u8In.Close()
	if err != nil {
		return nil, err
	}

	if listType != m3u8.MEDIA {
		return nil, errors.New("Please provide media file")
	}

	return pl.(*m3u8.MediaPlaylist), nil
}

/*func cbcDecrypt(in io.Reader, out io.Writer, key, iv []byte) error {
	bufSize := 24 * 1024
	buf := make([]byte, bufSize)

	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	blockMode := cipher.NewCBCDecrypter(cipherBlock, iv)
	for {
		n, err := io.ReadFull(in, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}
		blockMode.CryptBlocks(buf, buf[:n])
		if _, err := out.Write(buf[:n]); err != nil {
			return err
		}

		//v, err1 := out.Write(buf[:n])
		//if err1 != nil {
		//	return err
		//}
		//fmt.Println(v)

		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return nil
		}

				//if err == nil {
				//	//n -= 4096
				//	blockMode.CryptBlocks(buf, buf[:n])
				//	if _, err := out.Write(buf[:n]); err != nil {
				//		return err
				//	}
				//	copy(buf, buf[n:])
				//} else if err == io.ErrUnexpectedEOF {
				//	blockMode.CryptBlocks(buf, buf[:n])
				//	n -= int(buf[n-1])
				//	if _, err := out.Write(buf[:n]); err != nil {
				//		return err
				//	}
				//	return nil
				//} else if err == io.EOF{
				//
				//}
				//
				//	fmt.Println(n, err)

		//if _, err := out.Write(buf[:n]); err != nil {
		//	return err
		//}
		//copy(buf, buf[n:])

		//if err != nil {
		//	if err == io.ErrUnexpectedEOF || err == io.EOF {
		//		return nil
		//	}
		//	return err
		//}
	}
}*/

/*type cbcReader struct {
	rd  io.Reader
	tmp []byte // store unaligned cipher data
	buf []byte // store remaining unread plain data

	len int64 // expected length

	blockMode cipher.BlockMode
	bSize     int
}

// must be multiple of block size
func (r *cbcReader) read(p []byte) (n int, err error) {
	if len(p)%r.bSize != 0 {
		panic("A multiple of block size expected.")
	}
	n = copy(p, r.tmp)

	upstreamLen, err := r.rd.Read(p[n:])
	n += upstreamLen

	r.tmp = r.tmp[:n%r.bSize]
	n = floor(n, r.bSize)
	r.blockMode.CryptBlocks(p, p[:n])

	// store the trailing data
	copy(r.tmp, p[n:])

	r.len -= int64(n)
	return
}

func (r *cbcReader) Read(p []byte) (n int, err error) {
	//if len(p)%r.bSize == 0 {
	//	// use p directly
	//	n, err = r.read(p)
	//}
	// copy old data as much as possible
	n = copy(p, r.buf)
	p = p[n:] //remaining space
	if r.len == 0 || len(p) == 0 {
		// buf is enough, no more read needed
		copy(r.buf, r.buf[n:])
		r.buf = r.buf[:len(r.buf)-n]
		if r.len == 0 && n == 0 {
			return 0, io.EOF
		}
		return
	}

	nc := ceil(len(p), r.bSize)
	buf := make([]byte, nc)
	nc, err = r.read(buf)

	if r.len == 0 {
		// end reached
		// we should undo the padding
		// using PKCS7 here
		padLen := int(buf[nc-1])
		nc -= padLen
	}

	n2 := copy(p, buf[:nc])
	r.buf = r.buf[:nc-n2]
	copy(r.buf, buf[n2:])
	n += n2

	return
}

/*func (r *cbcReader) Read(out []byte) (int, error) {
	blockSize := r.blockMode.BlockSize()
	buf := make([]byte, ceil(len(out), blockSize))

	copy(buf, r.tmp)
	n, err := r.src.Read(buf[len(r.tmp):])

	n += len(r.tmp)

	// n >= blockSize
	end := floor(n, blockSize)
	r.blockMode.CryptBlocks(buf, buf[:end])
	copy(out, buf)

	// store the trailing data
	r.tmp = r.tmp[:n%blockSize]
	copy(r.tmp, buf[end:])

	// exactly multiple of blocks, so it's probably the last block
	if len(r.tmp) == 0 {
		n, err := r.src.Read(r.tmp[:blockSize])
		fmt.Println(n, err)
		if err == io.EOF {
			padLen := int(buf[end-1])
			return end - padLen, err
		} else if err != nil {
			return end, err
		}
		r.tmp = r.tmp[:n]
	}

	if err != nil {
		return end, err
	}

	return end, nil
}*/

/*func (r *cbcReader) Read(out []byte) (int, error) {
	blockSize := r.blockMode.BlockSize()

	// we can use out slice as a scratch space
	copy(out, r.tmp)
	n, err := r.src.Read(out[len(r.tmp):])

	if err == io.EOF {
		// end reached
		// we should undo the padding
		// using PKCS7 here
		return n, io.EOF
	}

	n += len(r.tmp)

	if n < blockSize {
		// we cannot decrypt such little data, caller should try again
		return 0, nil
	}
	// n >= blockSize
	end := floor(n, blockSize)
	r.blockMode.CryptBlocks(out, out[:end])

	// store the trailing data
	r.tmp = r.tmp[:n%blockSize]
	copy(r.tmp, out[end:])

	if err == io.EOF {
		// end reached
		// we should undo the padding
		// using PKCS7 here
		padLen := int(out[end-1])

		return end - padLen, err
	} else if err != nil {
		return end, err
	}

	return end, nil
}*/

/*func newCBCReader(r io.Reader, length int64, key []byte, iv []byte) (*cbcReader, error) {
	if length%int64(len(key)) != 0 {
		panic("Invalid input size")
	}
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockMode := cipher.NewCBCDecrypter(cipherBlock, iv)
	return &cbcReader{
		rd:        r,
		tmp:       make([]byte, 0, blockMode.BlockSize()),
		buf:       make([]byte, 0, blockMode.BlockSize()),
		len:       length,
		blockMode: blockMode,
		bSize:     blockMode.BlockSize(),
	}, nil
}*/
