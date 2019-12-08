package main

import (
	"fmt"
	"net/http"
)

func handler(rw http.ResponseWriter, r *http.Request) {
	if sz < 0 && err != nil {
		fmt.Println("write msg err")
	}
	target := url.URL.Parse("http://127.0.0.1:8081")
	reverseProxy := httputil.NewSingleReverseProxy(target)
	reverseProxy.ServeHTTP(rw, r)
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
