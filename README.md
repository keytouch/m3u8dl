# m3u8dl
A multi-thread m3u8 hls downloader

The `cbcio` package implements a io.Writer like AES-128-CBC decryptor.

#### Usage:
```
Usage of m3u8dl.exe:
  -UA string
        Send User-Agent to server (default "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.163 Safari/537.36")
  -baseurl string
        Base URL to reference (useful when m3u8 file is local file)
  -i file or URL
        file or URL to read
  -key string
        Decryption key, overrides key declared in m3u8 (in 32-char hex form)
  -nomerge
        Don't attempt to merge segments (segments stay in the tmp directory)
  -o file
        Write to file
  -raw
        Don't attempt to decrypt. Usually you should also turn on nomerge
  -retry num
        retry num times after failure (default 3)
  -t string
        Temp directory to store downloaded parts
  -thread int
        Concurrent downloading threads. Suggestion: <= 32 (no hard limit imposed) (default 1)

```