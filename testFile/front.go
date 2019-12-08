package main

import (
	"fmt"
	"net/http"
	"url"
)

func handler(rw http.ResponseWriter, r *http.Request) {
	newUrl, err := url.Parse("http://127.0.0.1:8081")
	if err != nil {
		log.Error("parse error")
	}

}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
