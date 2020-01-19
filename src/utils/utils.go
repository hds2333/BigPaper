package utils

import (
	"bytes"
	"fmt"
	"github.com/dennis/http"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	tmp = "/tmp"
	IPFS = "/usr/local/bin/ipfs"
)

func CalCidByContent(rw http.ResponseWriter, r *http.Request) (string, []byte) {
	log.Println("ContentLength:", r.ContentLength)
	//
	bodyReader := r.Body
	var buffer []byte
	buffer, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		log.Fatal("buffer error")
	}

	//rewind the request body
	r.Body = ioutil.NopCloser(bytes.NewReader(buffer))

	//File implements io.ReadCloser
	file, _, err := r.FormFile("uploadFile")
	if err != nil {
		log.Fatal("Get file")
	}

	defer file.Close()
	//Read file into bytes array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("Read file")
	}

	//Produce a filepath
	fileName := RandToken(12)
	newPath := filepath.Join(tmp, fileName)
	newFile, err := os.Create(newPath)
	if err != nil {
		log.Fatal("cant create new file", newPath)
	}

	defer newFile.Close()
	//write file into newpath
	if _, err := newFile.Write(fileBytes); err != nil {
		log.Fatal("cant write file")
	}
	//run the command and get the cid
	cmd := exec.Command(IPFS, "add", "-n", newPath)
	out, _ := cmd.Output()
	outslice := strings.Fields(string(out))
	return outslice[1], buffer
}

func RenderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

func RandToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

