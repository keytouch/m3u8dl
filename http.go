package main

import (
	"net/http"
)

var httpClient = &http.Client{}

// generic function to do a http get
// with UA, headers, etc. initialized
func httpGet(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", *flagUA)
	return httpClient.Do(req)
}
