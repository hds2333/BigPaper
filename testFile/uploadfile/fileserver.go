// curl -X POST -H "Content-Type: application/octet-stream" --data-binary '@filename' http://127.0.0.1:5050/upload

package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	buffer, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("result", buffer, 0600)
	filetype := http.DetectContentType(buffer)
	log.Println(filetype)
}

func main() {
	http.HandleFunc("/upload", uploadHandler)
	http.ListenAndServe(":5050", nil)
}
