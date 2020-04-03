package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-redis/redis"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

//author:Dennis Huang
type AdjItem struct {
	Policy int    `json:"policy"`
	Cid    string `json:"cid"`
	Delta  int    `json:"delta"`
}

const (
	IPFS     = "/usr/local/bin/ipfs"
	tmp      = `E:\BigPaper\src\data_node/tmp`
	keyNodes = "all-nodes"
	keyCids  = "all-cids"
	homepath  = `E:\BigPaper\src\data_node`
)

var client *redis.Client

func RandToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func CalCidByContent(rw http.ResponseWriter, r *http.Request) (string, []byte) {
	bodyReader := r.Body
	var buffer []byte
	buffer, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		log.Fatal("buffer error")
	}

	//rewind the request body
	r.Body = ioutil.NopCloser(bytes.NewReader(buffer))

	//Produce a filepath
	fileName := RandToken(12)
	newPath := filepath.Join(tmp, fileName)
	newFile, err := os.Create(newPath)
	if err != nil {
		log.Fatal(err)
	}

	defer newFile.Close()
	//write file into newpath
	if _, err := newFile.Write(buffer); err != nil {
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
	filename := RandToken(12)
	newFilePath := filepath.Join(tmp, filename)
	newFileHdl, err := os.Create(newFilePath)
	errPanic(err)
	sz, err := io.Copy(newFileHdl, r.Body)
	errPanic(err)
	log.Println("cp size: ", sz)
	cid := addOp(newFilePath)
	log.Println("Add replica on HOST", r.URL.Host)
	//if IsIPv4(r.URL.Host) {
		_, err = client.SAdd(cid, r.URL.Host).Result()
		panic(err)
	//} else {
	//	panic(errors.New("Wrong Format of IPv4 Addr"))
	//}
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
	//if utils.IsIPv4(r.URL.Host) {
	client.SRem(adjItem.Cid, r.URL.Host)
	//}
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
	} else if 2 == adjItem.Policy && adjItem.Delta < 0 { // to reduce replica
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
	cid, recoveredBody := CalCidByContent(rw, r)
	log.Println(cid)
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

	filename := RandToken(12)+"."+fileType[0]
	err = ioutil.WriteFile(filepath.Join(tmp, filename), buffer, 0644)
	errPanic(err)

	_, err = rw.Write([]byte("Success"))
	log.Fatal(err)
}

func errPanic(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	//return a handle struct
	fs := http.FileServer(http.Dir(homepath))

	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password:	"",
		DB:	0,
	})
	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		_, err := io.WriteString(w, "Hello, world!\n")
		if err != nil {
			log.Fatal(err)
		}
	}

	http.HandleFunc("/hello", helloHandler)

	http.Handle("/get", http.StripPrefix("get", fs))
	//route for the put file
	http.HandleFunc("/addreplica", AddReplica)
	http.HandleFunc("/adjreplica", AdjReplica)
	http.HandleFunc("/delreplica", DelReplica)
	http.HandleFunc("/put", Put)
	
	fmt.Println("sever start")
	err := http.ListenAndServe("127.0.0.1:28002", nil)
	if err != nil {
		log.Fatal("server down")
	}
}
