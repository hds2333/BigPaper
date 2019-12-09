package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	//"strconv"
	"strings"
	"time"
)

//proxy node has two task:
//1. transport the request to ipfs
//2. Periodicly send the replica adjust command to ipfs
const (
	keyRNTable    = "replica-node"
	keyNodes      = "all-nodes"
	KeyCids       = "all-cids"
	maxUploadSize = 150 * 1024 * 1024
	tmp           = "/tmp"
	IPFS          = "/usr/local/bin/ipfs"
)

var client *redis.Client

//calculate the cid of uploaded fiel
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

func reverseHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("Delay request")
	cid, bodyBuf := CalCidByContent(rw, r)
	log.Println("cid:", cid)
	rdr1 := ioutil.NopCloser(bytes.NewBuffer(bodyBuf))
	r.Body = rdr1

	//根据节点信用去转发请求
	newUrlStr := strings.Join([]string{"http://", "127.0.0.1:28002"}, "")
	remote, err := url.Parse(newUrlStr)
	if err != nil {
		log.Fatal("Parse URL")
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(rw, r)
}

type AdjItem struct {
	Policy int    `policy`
	Cid    string `cid`
	Delta  int    `delta`
}

//a period task
func AdjustReplica() {
	for {
		cids, err := client.SMembers(keyCids).Result()
		if err != nil {
			log.Println("get cids error")
		}
		//scan all the file, get the set of
		//adjustment & scan all the item, send request to node
		//select a node including the replica
		for _, cid := range cids {
			key := "adjust" + ":" + cid
			res, err := client.HMGet(key, "policy", "delta").Result()
			adjItem := AdjItem{
				Policy: res[0].(int),
				Delta:  res[1].(int),
			}

			if err != nil {
				panic(err)
			}

			busyNodes, _ := client.SMembers(cid).Result()
			desNode := getAHealthyNode(busyNodes)
			if item.Policy != 0 || item.Delta != 0 {
				sendAdjRequest(desNode, &item)
			}
		}
		time.Sleep(time.Hour)
	}
}

//create a http request to host
//send the command specified in json to host
//then host relay the request to final node with replica taken
func sendAdjRequest(host string, item *AdjItem) {
	httpClient := &http.Client{}
	jsonBytes, err := json.Marshal(*item)
	req, err := http.NewRequest(http.MethodPost, host, bytes.NewBuffer(jsonBytes))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-type", "application/json")
	resp, err := httpClient.Do(req)

	if err != nil {
		fmt.Println("send request err")
	}

	log.Println(resp.Status)
}

func randToken() string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func main() {
	http.HandleFunc("/", reverseHandler)
	http.ListenAndServe("127.0.0.1:28001", nil)
}
