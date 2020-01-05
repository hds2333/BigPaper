package main

import (
	"bytes"
	"fmt"
	"log"
	"io"
	"mime/multipart"
	"net/http"
//	"net/url"
	"os"
)

func err_exit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	filename := "/home/dennishuang/下载/537829220.jpg"
	file, err := os.Open(filename)
	defer file.Close()

	err_exit(err)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("uploadFile", "537829220.jpg")

	_, err = io.Copy(part, file)
	//_, err := io.Copy(part, file)
	err_exit(err)

	//parsedUrl, err := url.Parse("http://localhost:8099/upload")
	if err != nil {
		log.Fatal(err)
	}

	//create
	req, err := http.NewRequest("POST",
		"http://localhost:8099/upload", body)
	if req == nil {
		log.Fatal(err, "create request")
	}
	req.Header.Add("Content-Type",
		writer.FormDataContentType())
	req.Header.Add("Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Cache-Control", "max-age=0")
	req.Header.Add("User-Agent",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.77 Safari/537.36")
	req.Header.Add("Dnt", "1")
	req.Header.Add("Origin", "null")
	req.Header.Add("Upgrade-Insecure-Requests", "1")

	client := &http.Client{}
	fmt.Println(req.Header)
	resp, err := client.Do(req)
	log.Println(resp.Status)
	if err != nil {
		log.Fatal(err, resp.Status)
	}
