package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-redis/redis"
	//	"archive/tar"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type AdjItem struct {
	Policy int    `json:"policy"`
	Cid    string `json: "cid"`
	Delta  int    `json: "delta"`
}

const (
	IPFS = "/usr/local/bin/ipfs"
	tmp = "/tmp"
	KeyNodes = "all-nodes"
)
var client redis.Client

func AddOp(cid string) {
	cmd := exec.Command(IPFS, "tar", "add", cid)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func DelOp(cid string) {
	cmd := exec.Command(IPFS, "pin", "rm", cid)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	cmd = exec.Command(IPFS, "repo", "gc")
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

const maxUploadSize = 150 * 1024 * 1024
const uploadPath = "/tmp/ipfs"

func SendAddReplica(host string, adjItem *AdjItem) {
	/**currRepNum, err := client.SCard(cid).Result()
	if err != nil {
		log.Fatal(err)
	}**/

	delta, cid := adjItem.Delta, adjItem.Cid
	//make it a tar

	//get cid and op string,actually op string is unessary
	for i := 0; i < delta; i++ {

	}
}

func SendDelReplica(rw http.ResponseWriter, r *http.Request) {

}

func AdjReplica(rw http.ResponseWriter, r *http.Request) {
	bodyBuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal("read body")
	}

	var adjItem AdjItem
	err = json.Unmarshal(bodyBuf, &adjItem)
	if err != nil {
		log.Fatal("decode json err")
	}
	if adjItem.Policy == 0 && adjItem.Delta > 0 { // to increase replica
		//send replica &

	} else if adjItem.Policy == 1 && adjItem.Delta < 0 { // to reduce replica

	} else if adjItem.Policy == 0 { //erasure coding

	}
}

//user interface: used to upload file
//body: io.ReadCloser
//file: multipart.File
func Put(rw http.ResponseWriter, r *http.Request) {
	//isHealth := IsRequestHealthy(r)
	//r.Body = http.MaxBytesReader(rw, r.Body, maxUploadSize)
	cid, recoveredBody := CalCidByContent(rw, r)
	r.Body = ioutil.NopCloser(bytes.NewReader(revoveredBody))
	err := client.SAdd(keyCids, cid)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		log.Println("parse error", err)
		renderError(rw, "File too big", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("uploadFile")
	if err != nil {
		fmt.Println("FormFile err")
		renderError(rw, "FormFile error", http.StatusBadRequest)
		return
	}

	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	log.Println("body size:", len(fileBytes))

	if err != nil {
		fmt.Println("Read File error")
		renderError(rw, "Read File error", http.StatusBadRequest)
		return
	}

	fileName := randToken(12)
	newPath := filepath.Join(uploadPath, fileName)
	newFile, err := os.Create(newPath)
	if err != nil {
		fmt.Println(newPath)
		renderError(rw, "Cant create new file", http.StatusInternalServerError)
		return
	}

	defer newFile.Close()
	if _, err := newFile.Write(fileBytes); err != nil {
		renderError(rw, "Cant write file", http.StatusInternalServerError)
		return
	}

	//create a file use ipfs interface
	rw.Write([]byte("Success"))
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}

func randToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func main() {
	//return a handle struct
	fs := http.FileServer(http.Dir("/tmp"))
	http.Handle("/get", http.StripPrefix("get", fs))
	//route for the put file
	http.HandleFunc("/put", Put)
	http.ListenAndServe("127.0.0.1:28002", nil)
}
