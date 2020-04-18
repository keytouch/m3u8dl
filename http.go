package main

import (
	"net/http"
)

var httpClient = &http.Client{}

func initHttpClient() {
	//httpClient.Jar=cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
}

func httpGet(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", UA)
	return httpClient.Do(req)
}
