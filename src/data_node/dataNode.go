package main

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-redis/redis"

	"io/ioutil"
	"log"
	"net/http"
	//myhttp "github.com/dennis/http"
	"os"
	"os/exec"
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
	keyNodes = "all-nodes"
	keyCids  = "all-cids"
)

var client redis.Client

func addOp(filePath string) string {
	cmd := exec.Command(IPFS, "add", "-n", filePath)
	outputBytes, err  := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	outslice := strings.Fields(string(outputBytes))
	return outslice[1]
}

func delOp(cid string) {
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

const uploadPath = "/tmp/ipfs"

//SendAddRequest send request of increment replica to destiny node
func SendAddRequest(host string, adjItem *AdjItem) {
	cid := adjItem.Cid
	fileName := cid+".idrm"
	filePath := path.Join(uploadPath, fileName)
	cmdline := "ipfs cat " + cid + " > " + filePath
	cmd := exec.Command(cmdline)
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	fileHdl, err := os.Open(filePath)
	errPanic(err)

	resp, err := http.Post("http://localhost.com:5009/addrepica",
		"binary/octet-stream", fileHdl)
	if nil == err {
		log.Println(resp.Status)
	} else {
		log.Fatal("send request error: ", resp.Status)
	}
}

func SendDelRequest(host string, adjItem *AdjItem) {
	buffer, err := json.Marshal(*adjItem)
	if err != nil {
		panic(err)
	}
	body := bytes.NewReader(buffer)
	resp, err := http.Post(host,
		"application/json; charset=utf-8", body)
	if err != nil {
		log.Println(resp.Status)
		panic(err)
	}
}

//AddReplica handler of request to add replica
func AddReplica(rw http.ResponseWriter, r *http.Request) {
	filename := utils.RandToken(12)
	newFilePath := filepath.Join(tmp, filename)
	newFileHdl, err := os.Create(newFilePath)
	errPanic(err)
	sz, err := io.Copy(newFileHdl, r.Body)
	errPanic(err)
	log.Println("cp size: ", sz)
	cid := addOp(newFilePath)
	log.Println("Add replica on HOST", r.URL.Host)
	if utils.IsIPv4(r.URL.Host) {
		_, err := client.SAdd(cid, r.URL.Host).Result()
		panic(err)
	} else {
		panic(errors.New("Wrong Format of IPv4 Addr"))
	}
}

//DelReplica handler of request to del replica
func DelReplica(rw http.ResponseWriter, r *http.Request) {
	var adjItem AdjItem
	buffer, err := ioutil.ReadAll(r.Body)
	errPanic(err)
	err = json.Unmarshal(buffer, &adjItem)
	errPanic(err)
	delOp(adjItem.Cid)
	log.Println("Del replica on HOST: ", r.URL.Host)
	if utils.IsIPv4(r.URL.Host) {
		client.SRem(adjItem.Cid, r.URL.Host)
	}
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
	nodes, err := client.SDiff(adjItem.Cid, keyNodes).Result()
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
	cid, recoveredBody := utils.CalCidByContent(rw, r)
	r.Body = ioutil.NopCloser(bytes.NewReader(recoveredBody))
	client.SAdd(keyCids, cid)

	//create a file use ipfs interface
	buffer, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	mimeType := http.DetectContentType(buffer)
	fileType, err := mime.ExtensionsByType(mimeType)
	errPanic(err)

	filename := utils.RandToken(12)+"."+fileType[0]
	err = ioutil.WriteFile(filepath.Join(tmp, filename), buffer, 0644)
	errPanic(err)

	rw.Write([]byte("Success"))
}

func errPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	//return a handle struct
	fs := http.FileServer(http.Dir("/tmp"))
	http.Handle("/get", http.StripPrefix("get", fs))
	//route for the put file
	http.HandleFunc("/addreplica", AddReplica)
	http.HandleFunc("/adjreplica", AdjReplica)
	http.HandleFunc("/delreplica", DelReplica)
	http.HandleFunc("/put", Put)
	err := http.ListenAndServe("127.0.0.1:28002", nil)
	if err != nil {
		log.Fatal("server down")
	}
}
