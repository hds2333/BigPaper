package utils

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"github.com/dennis/http"
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
	fileName := randToken(12)
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
	cid := outslice[1]
	return cid, buffer
}

