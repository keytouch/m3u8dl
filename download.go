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
	queueLenDelta   = 16
	aes128BlockSize = 16
)

type job struct {
	id      uint64
	url     string
	key, iv []byte
}

func download(pl *m3u8.MediaPlaylist) error {
	queueLen := *flagThread + queueLenDelta
	jobs := make(chan job, queueLen)
	var wg sync.WaitGroup
	wg.Add(*flagThread)
	for i := 0; i < *flagThread; i++ {
		go dlWorker(i+1, jobs, &wg)
	}

	var seqIDs []uint64
	var lastIV []byte
	lastKey := customKey

	for _, seg := range pl.Segments {
		if seg == nil {
			break
		}
		seqIDs = append(seqIDs, seg.SeqId)

		if key := seg.Key; !*flagRaw && customKey == nil && key != nil {
			if key.Method == "AES-128" {
				logger.Println("downloading key from:", key.URI)
				resp, err := httpGet(key.URI)
				if err != nil {
					return fmt.Errorf("fetch key failed: %w", err)
				}

				lastKey = make([]byte, aes128BlockSize)
				_, err = io.ReadFull(resp.Body, lastKey)
				if err != nil {
					resp.Body.Close()
					return fmt.Errorf("fetch key failed: %w", err)
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
		err := merge(seqIDs)
		if err != nil {
			return fmt.Errorf("merge failed: %w", err)
		}
	}

	return nil
}

func dlWorker(id int, jobs <-chan job, wg *sync.WaitGroup) {
	defer wg.Done()
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
		fmt.Fprintf(&logBuf, "worker [%d] downloading seg %d", id, seg.id)
		if seg.key != nil {
			fmt.Fprintf(&logBuf, " with key %x, iv %x", seg.key, seg.iv)
		}
		logger.Println(logBuf.String())

		retry := 0
		for ; retry <= *flagRetry; retry++ {
			if retry > 0 {
				logger.Printf("worker [%d] retrying seg %d for %d time(s)\n", id, seg.id, retry)
			}

			var segIn io.ReadCloser
			if utils.IsValidUrl(seg.url) {
				req, err := http.NewRequest("GET", seg.url, nil)
				if err != nil {
					logErr.Printf("download seg %d from %s error: %v\n", seg.id, seg.url, err)
					continue
				}
				req.Header.Set("User-Agent", *flagUA)
				resp, err := dlClient.Do(req)
				if err != nil {
					logErr.Printf("download seg %d from %s error: %v\n", seg.id, seg.url, err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					logErr.Printf("download seg %d from %s response not good\n", seg.id, seg.url)
					resp.Body.Close()
					continue
				}

				segIn = resp.Body
			} else {
				var err error
				segIn, err = os.Open(seg.url)
				if err != nil {
					logErr.Printf("open seg %d (%s) error: %v\n", seg.id, seg.url, err)
					continue
				}
			}

			out, err := os.Create(filepath.Join(*flagTmpDir, strconv.Itoa(int(seg.id))+tmpSfx))
			if err != nil {
				logErr.Printf("seg %d create temp file error: %v\n", seg.id, err)
				segIn.Close()
				continue
			}

			if seg.key != nil {
				w := cbcio.NewWriter(out, seg.key, seg.iv)
				_, err := io.Copy(w, segIn)
				if err != nil {
					logErr.Printf("seg %d decrypt/write error: %v\n", seg.id, err)
					segIn.Close()
					out.Close()
					continue
				}

				if err := w.Final(); err != nil {
					logErr.Printf("seg %d decrypt/final error: %v\n", seg.id, err)
					segIn.Close()
					out.Close()
					continue
				}
			} else {
				_, err := io.Copy(out, segIn)
				if err != nil {
					logErr.Printf("seg %d write error: %v\n", seg.id, err)
					segIn.Close()
					out.Close()
					continue
				}
			}

			segIn.Close()
			out.Close()
			break
		}
		if retry > *flagRetry {
			// meaningless to continue
			logErr.Fatalf("seg %d failed after %d retries\n", seg.id, *flagRetry)
		}
	}
}

func merge(seqIDs []uint64) error {
	logger.Println("merging...")
	merged, err := os.Create(*flagOutput)
	if err != nil {
		return fmt.Errorf("create file error: %w", err)
	}
	defer merged.Close()

	for _, id := range seqIDs {
		tmpFilename := filepath.Join(*flagTmpDir, strconv.Itoa(int(id))+tmpSfx)
		tmpFile, err := os.Open(tmpFilename)
		if err != nil {
			return fmt.Errorf("open temp file error: %w", err)
		}

		_, err = io.Copy(merged, tmpFile)
		if err != nil {
			tmpFile.Close()
			return fmt.Errorf("write to merged file error: %w", err)
		}

		tmpFile.Close()
		os.Remove(tmpFilename)
	}
	return nil
}
