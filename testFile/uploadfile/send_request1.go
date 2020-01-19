package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	file, err := os.Open("/home/dennishuang/下载/537829220.jpg")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	res, err := http.Post("http://localhost:5050/upload", "binary/octet-stream", file)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	message, _ := ioutil.ReadAll(res.Body)
	fmt.Printf(string(message))
}
