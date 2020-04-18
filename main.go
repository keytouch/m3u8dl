package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logErr = log.New(os.Stderr, "", log.LstdFlags)
)
var (
	flagInput   = flag.String("i", "", "File or Url to read")
	flagOutput  = flag.String("o", "", "Destination to save the result")
	flagThread  = flag.Int("thread", 1, "Concurrent downloading threads. Suggestion: <= 32 (no hard limit imposed)")
	flagBaseUrl = flag.String("baseurl", "", "Base url to reference (useful when m3u8 file is local file)")
	flagKey     = flag.String("key", "", "Decryption key, overrides key declared in m3u8 (in 32-char hex form)")
)

func main() {
	//f, err := os.Open("media_b1973000_1.ts")
	/*f, err := os.Open(`D:\tmp\test-enc.bin`)
	if err != nil {
		logErr.Fatalln(err)
	}

	key, _ := hex.DecodeString("5E0ECC501DBC368679689947B5E8D5E1")
	iv, _ := hex.DecodeString("00000000000000000000000000000001")

	out, err := os.Create("test1.ts")
	if err != nil {
		logErr.Fatalln(err)
	}

	w := cbcio.NewCBCWriter(out, key, iv)
	fmt.Println(io.CopyBuffer(w, f, make([]byte, 8)))
	fmt.Println(w.Flush())
	//io.CopyBuffer(out, r, buf)
	f.Close()
	out.Close()

	return*/

	flag.Parse()
	/*keyBin, err := hex.DecodeString(*key)
	if err != nil {
		logErr.Fatalln(err)
	}*/
	if *flagBaseUrl == "" {
		*flagBaseUrl = (*flagInput)[:strings.LastIndex(*flagInput, "/")+1]
		fmt.Println(*flagBaseUrl)
	}

	pl, err := decode(*flagInput)
	if err != nil {
		logErr.Fatalln(err)
	}

	download(pl)
}
