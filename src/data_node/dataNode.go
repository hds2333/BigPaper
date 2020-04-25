package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

//author:Dennis Huang
type AdjItem struct {
	Policy int    `json:"policy"`
	Cid    string `json:"cid"`
	Delta  int    `json:"delta"`
}

const (
	IPFS     = "/usr/local/bin/ipfs"
	tmp      = "/tmp"
	keyNodes = "all-nodes"
	keyCids  = "all-cids"
	homepath = "."
)

var client *redis.ClusterClient

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
	outputBytes, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	outslice := strings.Fields(string(outputBytes))
	cmd1 := exec.Command(IPFS, "pin", "add", outslice[1])
	if err := cmd1.Run(); err != nil {
		log.Fatal("pin add:", err)
	}
	return outslice[1]
}

func delOp(cid string) {
	cmd := exec.Command(IPFS, "pin", "rm", cid)
	err := cmd.Run()
	if err != nil {
		log.Fatal("rm pin: ", err)
	}

	cmd = exec.Command(IPFS, "repo", "gc")
	err = cmd.Run()
	if err != nil {
		log.Fatal("repo gc: ", err)
	}
}

const uploadPath = "/tmp/ipfs"

//SendAddRequest send request of increment replica to destiny node
func SendAddRequest(dstIP string, cid string) int {
	fileName := cid + ".idrm"
	filePath := path.Join(uploadPath, fileName)
	fileHdl, err := os.Create(filePath)
	defer fileHdl.Close()
	errPanic(err)

	cmd := exec.Command("ipfs", "cat", cid)
	if out, err := cmd.Output(); err != nil {
		errPanic(err)
	} else {
		if _, err := fileHdl.Write(out); err != nil {
			errPanic(err)
		}
	}
	filePt, err := os.Open(filePath)
	defer filePt.Close()
	if err != nil {
		log.Fatal(err)
	}
	//port := "28002"
	dstUrl := "http://" + dstIP + ":28002" + "/addreplica"
	resp, err := http.Post(dstUrl, "application/octet-stream", filePt)
	if err == nil {
		log.Println("Add Replica Response: ", resp.Status)
	} else {
		log.Fatal(err, resp.Status)
	}
	return resp.StatusCode
}

func SendDelRequest(dstIP string, adjItem *AdjItem) int {
	buffer, err := json.Marshal(*adjItem)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewReader(buffer)

	dstUrl := "http://" + dstIP + ":28002" + "/delreplica"
	resp, err := http.Post(dstUrl, "application/json; charset=utf-8", body)

	if err == nil {
		log.Println("Del Replica Response: ", resp.Status)
	} else {
		log.Fatal(err, resp.Status)
	}

	return resp.StatusCode
}

//AddReplica handler of request to add replica
func AddReplica(rw http.ResponseWriter, r *http.Request) {
	log.Println("Add replica")
	filename := RandToken(12)
	newFilePath := filepath.Join(tmp, filename)
	newFileHdl, err := os.Create(newFilePath)
	errPanic(err)

	_, err = io.Copy(newFileHdl, r.Body)
	errPanic(err)

	cid := addOp(newFilePath)
	//if IsIPv4(r.URL.Host) {
	f := func(c rune) bool {
		return c == ':'
	}
	IPAddr := strings.FieldsFunc(r.Host, f)[0]
	log.Println("Add replica on HOST: ", IPAddr)
	if len(IPAddr) > 0 {
		_, err = client.SAdd(cid, IPAddr).Result()
		if err != nil {
			log.Fatal(err)
		}
	}
	//} else {
	//	panic(errors.New("Wrong Format of IPv4 Addr"))
	//}
	rw.WriteHeader(200)
}

//DelReplica handler of request to del replica
func DelReplica(rw http.ResponseWriter, r *http.Request) {
	log.Println("Del Replica")
	var adjItem AdjItem
	buffer, err := ioutil.ReadAll(r.Body)
	errPanic(err)
	err = json.Unmarshal(buffer, &adjItem)
	if err != nil {
		log.Fatal(err, "Decode Json Error")
	}
	//if utils.IsIPv4(r.URL.Host) {
	f := func(c rune) bool {
		return c == ':'
	}
	IPAddr := strings.FieldsFunc(r.Host, f)[0]

	log.Println("Del Replica on Host: ", IPAddr)
	affectedLine, err := client.SRem(adjItem.Cid, IPAddr).Result()
	if err != nil {
		log.Fatal("Redis Rm Err: ", err)
	} else {
		log.Printf("%d line deleted", affectedLine)
	}
	if affectedLine > 0 {
		delOp(adjItem.Cid)
	}

	/*
		flag, err := client.Exists(adjItem.Cid).Result()
		if err != nil {
			log.Fatal("Check Redis State", err)
		} else {
			if flag == 0 {
				_, err = client.SRem(keyCids, adjItem.Cid).Result()
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	*/
	//}
	rw.WriteHeader(200)
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
	log.Printf("我是本次副本调节员:%s\n", r.Host)
	log.Println("recv json: ", adjItem)
	//存储的是没有存储CID的节点，为了得到这些节点
	//需要用存储CID的节点集合和节点表作交集
	//nodes, err := client.SDiff(adjItem.Cid, keyNodes).Result()
	has, err := client.SMembers(adjItem.Cid).Result()
	if err != nil {
		log.Fatal("Smem Err: ", err)
	}
	all, err := client.SMembers(keyNodes).Result()
	if err != nil {
		log.Fatal("Smem Err: ", err)
	}
	nodes := difference(all, has)
	if 1 == adjItem.Policy && adjItem.Delta > 0 { // to increase replica
		if len(nodes) > adjItem.Delta {
			nodes = nodes[0:adjItem.Delta]
		}
		log.Printf("即将增加副本在节点:%+v\n", nodes)
		for i := 0; i < len(nodes); i++ {
			SendAddRequest(nodes[i], adjItem.Cid)
		}
	} else if 2 == adjItem.Policy && adjItem.Delta < 0 { // to reduce replica
		delta := int(math.Abs(float64(adjItem.Delta)))
		//occupiedNodes is used to represent nodes having cid
		occupiedNodes, err := client.SMembers(adjItem.Cid).Result()
		if err != nil {
			log.Fatal("SMembers Error")
		}
		log.Printf("[Delete Replica] delta:%d, occupiedNodes:%+v\n", delta, occupiedNodes)
		if len(occupiedNodes) > delta {
			occupiedNodes = occupiedNodes[0:delta]
		}
		log.Printf("即将删除副本在节点:%+v\n", occupiedNodes)
		for i := 0; i < delta; i++ {
			SendDelRequest(occupiedNodes[i], &adjItem)
		}
	} else if 0 == adjItem.Policy { //erasure coding
		occupiedNodes, err := client.SMembers(adjItem.Cid).Result()
		if err != nil {
			log.Fatal("SMembers Error")
		}
		log.Printf("即将删除副本在节点:%+v\n", occupiedNodes)
		for i := 0; i < len(occupiedNodes); i++ {
			SendDelRequest(occupiedNodes[i], &adjItem)
		}
		// TODO: create the erasure code file
	}
}

func difference(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	inter := intersect(slice1, slice2)
	for _, v := range inter {
		m[v]++
	}

	for _, value := range slice1 {
		times, _ := m[value]
		if times == 0 {
			nn = append(nn, value)
		}
	}
	return nn
}

func intersect(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times, _ := m[v]
		if times == 1 {
			nn = append(nn, v)
		}
	}
	return nn
}

//user interface: used to upload file
//body: io.ReadCloser
//file: multipart.File
func Put(rw http.ResponseWriter, r *http.Request) {
	//isHealth := IsRequestHealthy(r)
	cid, recoveredBody := CalCidByContent(rw, r)

	log.Println(cid, len(recoveredBody))

	r.Body = ioutil.NopCloser(bytes.NewReader(recoveredBody))

	log.Println("<------------ready to add:", cid, "--------------->")
	log.Println(client)
	res := client.SAdd(keyCids, cid)
	log.Println("after insert: ", res.String())
	if err := res.Err(); err != nil {
		log.Fatal(err)
	}
	//create a file use ipfs interface
	buffer, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errPanic(err)
	}
	mimeType := http.DetectContentType(buffer)
	log.Println("mime: ", mimeType)

	fileType, err := mime.ExtensionsByType(mimeType)
	errPanic(err)
	log.Println("filetype: ", fileType)

	filename := RandToken(12) + fileType[0]
	fpath := filepath.Join(tmp, filename)
	err = ioutil.WriteFile(fpath, buffer, 0644)
	errPanic(err)
	log.Println("write success: ", fpath)
	cmd := exec.Command("ipfs-cluster-ctl", "add", fpath)
	err = cmd.Run()
	errPanic(err)
	log.Println("add to cluster success")

	_, err = rw.Write([]byte("Success"))
	errPanic(err)
}

func errPanic(err error) {
	if err != nil {
		log.Fatal(err)
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
		log.Println(pong)
	}
}

func main() {
	var host_str string
	if len(os.Args) > 0 {
		host_str = os.Args[1]
	}

	//return a handle struct
	fs := http.FileServer(http.Dir(homepath))

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		_, err := io.WriteString(w, "Hello, world!\n")
		if err != nil {
			log.Fatal(err)
		}
	}

	http.HandleFunc("/hello", helloHandler)
	//route for the put file
	http.HandleFunc("/addreplica", AddReplica)
	http.HandleFunc("/adjreplica", AdjReplica)
	http.HandleFunc("/delreplica", DelReplica)
	http.HandleFunc("/put", Put)
	http.Handle("/get", fs)

	fmt.Println("sever start")
	url := host_str + ":28002"
	log.Println("URL: ", url)
	err := http.ListenAndServe(url, nil)
	if err != nil {
		log.Fatal(err)
	}
}
