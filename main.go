package main

import (
	"encoding/hex"
	"flag"
	"log"
	"os"
	"strings"
)

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.163 Safari/537.36"

var (
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logErr = log.New(os.Stderr, "", log.LstdFlags)
)
var (
	flagInput   = flag.String("i", "", "`file or URL` to read")
	flagOutput  = flag.String("o", "", "Write to `file`")
	flagTmpDir  = flag.String("t", "", "Temp directory to store downloaded parts")
	flagThread  = flag.Int("thread", 1, "Concurrent downloading threads. Suggestion: <= 32 (no hard limit imposed)")
	flagUA      = flag.String("UA", defaultUA, "Send User-Agent to server")
	flagBaseURL = flag.String("baseurl", "", "Base URL to reference (useful when m3u8 file is local file)")
	flagKey     = flag.String("key", "", "Decryption key, overrides key declared in m3u8 (in 32-char hex form)")
	flagRaw     = flag.Bool("raw", false, "Don't attempt to decrypt. Usually you should also turn on nomerge")
	flagNoMerge = flag.Bool("nomerge", false, "Don't attempt to merge segments (segments stay in the tmp directory)")
	flagRetry   = flag.Int("retry", 3, "retry `num` times after failure")
)
var customKey []byte

func main() {
	flag.Parse()

	if !*flagNoMerge && *flagOutput == "" {
		logErr.Fatalln("output file must be specified")
	}

	var err error
	if *flagKey != "" {
		customKey, err = hex.DecodeString(*flagKey)
		if err != nil {
			logErr.Fatalln("invalid key given:", err)
		}
		if len(customKey) != aes128BlockSize {
			logErr.Fatalln("please provide key in 32-char hex form")
		}
	}

	if *flagBaseURL == "" {
		*flagBaseURL = (*flagInput)[:strings.LastIndex(*flagInput, "/")+1]
	} else if (*flagBaseURL)[len(*flagBaseURL)-1] != '/' {
		*flagBaseURL = *flagBaseURL + "/"
	}
	logger.Println("using base URL:", *flagBaseURL)

	logger.Println("decoding m3u8 from:", *flagInput)
	pl, err := decode(*flagInput)
	if err != nil {
		logErr.Fatalln("decode m3u8 file failed:", err)
	}

	err = download(pl)
	if err != nil {
		logErr.Fatalln("download failed:", err)
	}
}
