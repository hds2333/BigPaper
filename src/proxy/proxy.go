package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/go-redis/redis"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"utils"

	//"strconv"
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
)

var client *redis.Client

func reverseHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("Delay request")
	_, bodyBuf := utils.CalCidByContent(rw, r)
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

type AdjItem struct {
	Policy int    `policy`
	Cid    string `cid`
	Delta  int    `delta`
}

//AdjustReplica a period task
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
			//TODO: 1. import training model
			//TODO: 2. judging from the heathy to select a node
			log.Println(cid)
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

func main() {
	go func() {
		AdjustReplica()
	}()
	http.HandleFunc("/", reverseHandler)
	http.ListenAndServe("127.0.0.1:28001", nil)
}
