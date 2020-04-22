package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"m3u8dl/cbcio"
	"m3u8dl/utils"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grafov/m3u8"
)

const (
	tmpSfx          = ".tmp"
	queueLen        = 32
	aes128BlockSize = 16
)

type job struct {
	id      uint64
	url     string
	key, iv []byte
}

func download(pl *m3u8.MediaPlaylist) {
	jobs := make(chan job, queueLen)

	var wg sync.WaitGroup
	wg.Add(*flagThread)
	for i := 0; i < *flagThread; i++ {
		go dlWorker(i+1, jobs, &wg)
	}

	var segLen int
	var lastKey, lastIV []byte
	if *flagKey != "" {
		var err error
		lastKey, err = hex.DecodeString(*flagKey)
		if err != nil {
			logErr.Fatalln(err)
		}
	}

	for _, seg := range pl.Segments {
		if seg == nil {
			continue
		}
		segLen++

		if key := seg.Key; !*flagRaw && *flagKey == "" && key != nil {
			if key.Method == "AES-128" {
				logger.Println("downloading key from:", key.URI)
				resp, err := httpGet(key.URI)
				if err != nil {
					logErr.Fatalln(err)
				}

				lastKey = make([]byte, aes128BlockSize)
				_, err = io.ReadFull(resp.Body, lastKey)
				if err != nil {
					logErr.Fatalln(err)
				}

				logger.Printf("got key: %x\n", lastKey)

				resp.Body.Close()
			} else {
				lastKey = nil
			}

			if key.IV == "" {
				lastIV = nil
			} else {
				var err error
				lastIV, err = hex.DecodeString(key.IV)
				if len(lastIV) != aes128BlockSize || err != nil {
					logErr.Println("bad iv, continue with sequence id based iv")
					lastIV = nil
				}
			}
		}

		seg := job{
			id:  seg.SeqId,
			url: *flagBaseURL + seg.URI,
			key: lastKey,
			iv:  lastIV,
		}
		if seg.key != nil && seg.iv == nil {
			seg.iv = make([]byte, aes128BlockSize)
			binary.BigEndian.PutUint64(seg.iv[8:], seg.id)
		}
		jobs <- seg
	}
	close(jobs)

	wg.Wait()

	if !*flagNoMerge {
		err := merge(segLen)
		if err != nil {
			logErr.Fatalln(err)
		}
	}
}

func dlWorker(id int, jobs <-chan job, wg *sync.WaitGroup) {
	dlClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	for seg := range jobs {
		var logBuf strings.Builder
		fmt.Fprintf(&logBuf, "worker [%d] downloading %d", id, seg.id)
		if seg.key != nil {
			fmt.Fprintf(&logBuf, " with key %x iv %x", seg.key, seg.iv)
		}
		logger.Println(logBuf.String())

		var segIn io.ReadCloser
		if utils.IsValidUrl(seg.url) {
			req, err := http.NewRequest("GET", seg.url, nil)
			if err != nil {
				logErr.Println(err)
				continue
			}
			req.Header.Set("User-Agent", *flagUA)
			resp, err := dlClient.Do(req)
			if err != nil {
				logErr.Println(seg.id, err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				logErr.Println("Response not good:", seg.id)
				resp.Body.Close()
				continue
			}

			segIn = resp.Body
		} else {
			var err error
			segIn, err = os.Open(seg.url)
			if err != nil {
				logErr.Println(seg.id, err)
				continue
			}
		}

		out, err := os.Create(filepath.Join(*flagTmpDir, strconv.Itoa(int(seg.id))+tmpSfx))
		if err != nil {
			logErr.Println(err)
			segIn.Close()
			continue
		}

		if seg.key != nil {
			w := cbcio.NewWriter(out, seg.key, seg.iv)
			_, err := io.Copy(w, segIn)
			if err != nil {
				logErr.Println(err)
				segIn.Close()
				continue
			}

			if err := w.Flush(); err != nil {
				logErr.Println(err)
				segIn.Close()
				continue
			}
		} else {
			_, err := io.Copy(out, segIn)
			if err != nil {
				logErr.Println(err)
				segIn.Close()
				continue
			}
		}

		segIn.Close()
		out.Close()
	}
	wg.Done()
}

func merge(segLen int) error {
	logger.Println("merging...")
	merged, err := os.Create(*flagOutput)
	if err != nil {
		return err
	}
	defer merged.Close()

	for i := 0; i < segLen; i++ {
		tmpFilename := filepath.Join(*flagTmpDir, strconv.Itoa(i)+tmpSfx)
		tmpFile, err := os.Open(tmpFilename)
		if err != nil {
			return err
		}

		_, err = io.Copy(merged, tmpFile)
		if err != nil {
			tmpFile.Close()
			return err
		}

		err = merged.Sync()
		if err != nil {
			tmpFile.Close()
			return err
		}

		tmpFile.Close()
		os.Remove(tmpFilename)
	}
	return nil
}
