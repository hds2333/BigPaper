package main

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
	keyNodes   = "all-nodes"
	keyCids    = "all-cids"
	tmp        = "/tmp"
	IPFS       = "/usr/local/bin/ipfs"
	predictLen = 5
)

var (
	periodIndex = 0
	modelMap    map[string][]int
	client      *redis.ClusterClient
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
	nodes, err := client.SMembers(keyNodes).Result()
	if err != nil {
		log.Fatal("Smember Err:", keyNodes)
	}
	//TODO host := SelectNode(nodes)
	host := nodes[0]
	newUrlStr := strings.Join([]string{"http://", host, ":28002"}, "")
	remote, err := url.Parse(newUrlStr)
	if err != nil {
		log.Fatal("Parse URL")
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(rw, r)
}

//AdjustReplica a period task
func AdjustReplica() {
	log.Println("进入工作状态")
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
			log.Printf("给单位注意,现在开始处理第[%s]号文件\n", cid)
			nodes, err := client.SMembers(cid).Result()
			if err != nil {
				log.Fatal("SMem err")
			}
			var policy, delta int
			currRepNum := len(nodes)
			delta = modelMap[cid][periodIndex] - currRepNum
			if modelMap[cid][periodIndex] < 2 {
				policy = 0
			} else if delta > 0 {
				policy = 1
			} else if delta < 0 {
				policy = 2
			}

			item := AdjItem{
				Cid:    cid,
				Policy: policy,
				Delta:  delta,
			}
			//TODO Add Nodes Selection Module
			if len(nodes) > 0 {
				sendAdjRequest(nodes[0], &item)
				log.Printf("我是代理,我向节点[%s]发送了命令%+v\n", nodes[0], item)
			} else {
				log.Printf("第[%s]号文件的副本已经全部删除，请手动增加:\n", cid)
				var fileName string
				fmt.Scan(&fileName)
				cmdline := exec.Command("/usr/local/bin/ipfs-cluster-ctl", "add", fileName)
				if err := cmdline.Run(); err != nil {
					log.Fatal("ipfs-cluster-ctl add error:", err)
				}
			}
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
	url := "http://" + host + ":28002" + "/adjreplica"
	req, err := http.NewRequest(http.MethodPost, url,
		bytes.NewBuffer(jsonBytes))
	if err != nil {
		panic(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal("send request err:", err)
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

func init() {
	client = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"172.22.0.2:7000",
			"172.22.0.2:7001",
			"172.22.0.2:7002",
			"172.22.0.2:7003",
			"172.22.0.2:7004",
			"172.22.0.2:7005",
		},
		Password: "",
	})
	pong, err := client.Ping().Result()
	if err != nil {
		log.Fatal("Redis Connect Err")
	} else {
		log.Println("Redis Connect Succeed", pong)
	}
	importModel("model")
}

func main() {
	go func() {
		AdjustReplica()
	}()

	http.HandleFunc("/", reverseHandler)
	err := http.ListenAndServe("127.0.0.1:28001", nil)
	if err != nil {
		log.Fatal("proxy listen")
	}
}
