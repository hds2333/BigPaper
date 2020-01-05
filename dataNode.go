package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"path"

	"github.com/go-redis/redis"

	//	"archive/tar"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	//"net/http"
	"github.com/dennis/http"
	"os"
	"os/exec"
	"path/filepath"
	"utils"
)

type AdjItem struct {
	Policy int    `json:"policy"`
	Cid    string `json: "cid"`
	Delta  int    `json: "delta"`
}

const (
	IPFS     = "/usr/local/bin/ipfs"
	tmp      = "/tmp"
	KeyNodes = "all-nodes"
	KeyCids  = "all-cids"
)

var client redis.Client

func AddOp(filePath string) {
	cmd := exec.Command(IPFS, "add", filePath)
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

func err_exit(err error, msg string) {
	log.Fatal(err)
}

//Processor for
func SendAddRequest(host string, adjItem *AdjItem) {
	//export the file from the IPFS repo
	cid := adjItem.Cid
	filePath := path.Join(uploadPath, cid, ".idrm")
	cmdline := "ipfs cat " + cid + " > " + filePath
	cmd := exec.Command(cmdline)
	if err := cmd.Run(); err != nil {
		log.Fatal("cmd.Run")
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal("open file")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("idrm", filePath)
	if err != nil {
		log.Fatal("create form-data body err")
	}

	sz, err := io.Copy(part, file)
	if err != nil && sz < 0 {
		log.Fatal(err)
	}

	//create a http request
	req, err := http.NewRequest("post", "http://localhost.com/addrepica", body)
	if err != nil {
		log.Fatal("new request")
	}
	//req.Header.Add("Content-Type", "multipart/form-data")
	client := http.DefaultClient
	if resp, err := client.Do(req); nil == err {
		log.Println(resp.Status)
	} else {
		log.Fatal("send request error: ", resp.Status)
	}
}

func SendDelRequest(host string, adjItem *AdjItem) {

}

//processor
func AddReplica(rw http.ResponseWriter, r *http.Request) {

}
//Processor
func DelReplica(rw http.ResponseWriter, r *http.Request) {

}

//Processor
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
	nodes, err := client.SDiff(adjItem.Cid, KeyNodes).Result()
	if err != nil {
		log.Fatal("datanode:AdjReplica:line99")
	}

	if 1 == adjItem.Policy && adjItem.Delta > 0 { // to increase replica
		nodes := nodes[0:adjItem.Delta]
		for i := 0; i < len(nodes); i++ {
			SendAddRequest(nodes[i], &adjItem)
		}
	} else if 1 == adjItem.Policy && adjItem.Delta < 0 { // to reduce replica
		nodes := nodes[0:adjItem.Delta]
		for i := 0; i < len(nodes); i++ {
			SendDelRequest(nodes[i], &adjItem)
		}
	} else if 0 == adjItem.Policy { //erasure coding
		for i := 0; i < len(nodes); i++ {
			SendDelRequest(nodes[i], &adjItem)
		}
		// TODO: create the erasure code file
	}
}

//user interface: used to upload file
//body: io.ReadCloser
//file: multipart.File
func Put(rw http.ResponseWriter, r *http.Request) {
	//isHealth := IsRequestHealthy(r)
	//r.Body = http.MaxBytesReader(rw, r.Body, maxUploadSize)
	cid, recoveredBody := utils.CalCidByContent(rw, r)
	r.Body = ioutil.NopCloser(bytes.NewReader(recoveredBody))
	client.SAdd(KeyCids, cid)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		log.Println("parse error", err)
		renderError(rw, "File too big", http.StatusBadRequest)
		return
	}

	file,_, err := r.FormFile("uploadFile")
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
	http.HandleFunc("/adjreplica", AdjReplica)
	http.HandleFunc("/addreplica", AddReplica)
	http.HandleFunc("/delreplica", DelReplica)
	http.HandleFunc("/put", Put)
	err := http.ListenAndServe("127.0.0.1:28002", nil)
	if err != nil {
		log.Fatal("server down")
	}
}
