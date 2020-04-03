package proxy

import (
	"bufio"
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
	"strconv"
	"strings"
	"time"
)

//proxy node has two task:
//1. transport the request to ipfs
//2. Periodicly send the replica adjust command to ipfs
const (
	keyNodes      = "all-nodes"
	keyCids       = "all-cids"
	tmp           = "/tmp"
	IPFS          = "/usr/local/bin/ipfs"
	predictLen = 72
)

var (
	periodIndex = 0
	modelMap map[string][]int
	client *redis.Client
)

type AdjItem struct {
	Policy int    `json:"policy"`
	Cid    string `json:"cid"`
	Delta  int    `json:"delta"`
}

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

func reverseHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("Delay request")

	_, bodyBuf := CalCidByContent(rw, r)
	//log.Println("cid:", cid)
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



//AdjustReplica a period task
func AdjustReplica() {
	for {
		cids, err := client.SMembers(keyCids).Result()
		if err != nil {
			log.Fatal(err)
		}
		//scan all the file, get the set of
		//adjustment & scan all the item, send request to node
		//select a node including the replica
		for _, cid := range cids {
			//TODO: judging from the heathy to select a node
			nodes, err := client.SMembers(cid).Result()
			if err != nil {
				log.Fatal("redis error")
			}
			var policy, delta int
			//3 is a temporary threshold
			if periodIndex > 0 {
				delta = modelMap[cid][periodIndex] -
					modelMap[cid][periodIndex-1]
				if  modelMap[cid][periodIndex] < 3 {
					policy = 0
				} else if delta > 0{
					policy = 1
				} else if delta < 0 {
					policy = 2
				}
			}

			item := AdjItem{
				Cid: cid,
				Policy: policy,
				Delta: delta,
			}
			sendAdjRequest(nodes[0], &item)
			log.Println(cid)
		}
		periodIndex++
		if periodIndex == predictLen {
			periodIndex = 0
		}
		time.Sleep(10 * time.Second)
	}
}

//create a http request to host
//send the command specified in json to host
//then host relay the request to final node with replica taken
func sendAdjRequest(host string, item *AdjItem) {
	httpClient := &http.Client{}
	jsonBytes, err := json.Marshal(*item)
	url := host + "/AdjRequest"
	req, err := http.NewRequest(http.MethodPost, url,
		bytes.NewBuffer(jsonBytes))
	if err != nil {
		panic(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal("send request err", err)
	}

	log.Println(resp.Status)
}

func importModel(modelName string) {
	modelMap = make(map[string][]int)
	file, err := os.Open(modelName)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		strs := strings.Fields(scanner.Text())
		var nums []int
		for i := 1; i < len(strs); i++ {
			num, err := strconv.Atoi(strs[i])
			if err != nil {
				log.Fatal("error format of model")
			}
			nums = append(nums, num)
		}
		modelMap[strs[0]] = nums
	}
}

func main() {
	client = redis.NewClient(&redis.Options{
		Addr:	"localhost:6379",
		Password:	"",
		DB:	0,
	})

	importModel("model")

	go func() {
		AdjustReplica()
	}()

	http.HandleFunc("/", reverseHandler)
	err := http.ListenAndServe("127.0.0.1:28001", nil)
	if err != nil {
		log.Fatal("proxy listen")
	}
}
