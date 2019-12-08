package main

import (
	"fmt"
	"net/http"
)

func handler(rw http.ResponseWriter, r *http.Request) {
	str := "hello world!"
	rw.Write([]byte(str))
	fmt.Println(r.URL.Host, " ", r.URL.Scheme, " ", r.URL.Path)
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8081", nil)
}
