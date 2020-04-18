package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/grafov/m3u8"
	"io"
	"m3u8dl/cbcio"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const queueLen = 32

//TODO
const UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.163 Safari/537.36"

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

		if key := seg.Key; *flagKey == "" && key != nil {
			if key.Method == "AES-128" {
				logger.Println("downloading key from:", key.URI)
				resp, err := httpGet(key.URI)
				if err != nil {
					logErr.Fatalln(err)
				}

				lastKey = make([]byte, 16)
				_, err = io.ReadFull(resp.Body, lastKey)
				if err != nil {
					logErr.Fatalln(err)
				}

				resp.Body.Close()
			} else {
				lastKey = nil
			}

			if key.IV == "" {
				lastIV = nil
			} else {
				var err error
				lastIV, err = hex.DecodeString(key.IV)
				if err != nil {
					logErr.Println("bad iv")
				}
			}
		}

		seg := job{
			id:  seg.SeqId,
			url: *flagBaseUrl + seg.URI,
			key: lastKey,
			iv:  lastIV,
		}
		if seg.key != nil && seg.iv == nil {
			seg.iv = make([]byte, 16)
			binary.BigEndian.PutUint64(seg.iv[8:], seg.id)
		}
		jobs <- seg
	}
	close(jobs)

	wg.Wait()
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

		req, err := http.NewRequest("GET", seg.url, nil)
		if err != nil {
			logErr.Println(err)
			continue
		}
		req.Header.Set("User-Agent", UA)
		resp, err := dlClient.Do(req)
		if err != nil {
			logErr.Println(seg.id, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logErr.Println("Response not good:", seg.id)
			continue
		}

		file, err := os.Create(*flagOutput + strconv.Itoa(int(seg.id)) + ".ts")
		if err != nil {
			logErr.Println(err)
			continue
		}

		if seg.key != nil {
			w := cbcio.NewWriter(file, seg.key, seg.iv)
			_, err := io.Copy(w, resp.Body)
			if err != nil {
				logErr.Println(err)
				continue
			}

			if err := w.Flush(); err != nil {
				logErr.Println(err)
				continue
			}
		} else {
			_, err := io.Copy(file, resp.Body)
			if err != nil {
				logErr.Println(err)
				continue
			}
		}

		resp.Body.Close()
		file.Close()
	}
	wg.Done()
}

func mergeWrite() {

}
