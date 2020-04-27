package main

import (
	"container/ring"
	"encoding/json"
	"github.com/go-redis/redis"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
)

type Metric struct {
	Name    string
	Data    float64
	Expired int
	Src     string
}

var (
	cpuBuffer   *ring.Ring
	memBuffer   *ring.Ring
	churnBuffer *ring.Ring
	client      *redis.ClusterClient
)

const keyCreditTable = "all-credit"

func AcceptMetics(rw http.ResponseWriter, r *http.Request) {
	var metric Metric
	jsonBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal("Read Metric Request Error", err)
	}
	err = json.Unmarshal(jsonBytes, &metric)
	if err != nil {
		log.Fatal("Decode Json Error")
	}
	if "cpu" == metric.Name {
		cpuBuffer.Value = metric
		cpuBuffer = cpuBuffer.Next()
	} else if "memory" == metric.Name {
		memBuffer.Value = metric
		memBuffer = memBuffer.Next()
	} else if "churn" == metric.Name {
		churnBuffer.Value = metric
		churnBuffer = churnBuffer.Next()
	}
}

func InitRing(r *ring.Ring) {
	nodes, err := client.SMembers("all-nodes").Result()
	if err != nil {
		log.Fatal("SMem all-nodes:", err)
	}
	ring.New(len(nodes))
}

func CalcGrade(n int) {
	mapSlice := make(map[string][3]float64)

	for k, v := range mapSlice {
		log.Println(k, v)
	}

	for i := 0; i < cpuBuffer.Len(); i++ {
		m := cpuBuffer.Value.(Metric)
		a := mapSlice[m.Src]
		a[0] = m.Data
	}

	for i := 0; i < memBuffer.Len(); i++ {
		m := memBuffer.Value.(Metric)
		log.Println(reflect.TypeOf(m))
		a := mapSlice[m.Src]
		a[1] = m.Data
	}

	for i := 0; i < churnBuffer.Len(); i++ {
		m := churnBuffer.Value.(Metric)
		a := mapSlice[m.Src]
		a[2] = m.Data
	}

	for k, v := range mapSlice {
		log.Printf("IP:%s,Metrics:%+v,%+v,%+v\n", k, v[0], v[1], v[2])
		grade := (v[0]/8.0 + v[1]/16384.0 + v[2]/20.0) * 100.0
		z := &redis.Z{
			Score:  grade,
			Member: k,
		}
		_, err := client.ZAdd(keyCreditTable, z).Result()
		if err != nil {
			log.Fatal("ZAdd Error:", err)
		}
	}
}

func SelectNodes(flag int) string {
	//acend
	var node string
	if 0 == flag {
		res, err := client.ZRange(keyCreditTable, 0, 0).Result()
		if err != nil {
			log.Fatal("ZRange Error", err)
		}
		if len(res) > 0 {
			log.Printf("Elected Max Node:%s\n", res[0])
			node = res[0]
		}
	} else if 1 == flag {
		res, err := client.ZRevRange(keyCreditTable, 0, 0).Result()
		if err != nil {
			log.Fatal("ZRange Error", err)
		}
		if len(res) > 0 {
			log.Printf("Elected Min Node:%s\n", res[0])
			node = res[0]
		}
	}
	return node
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
}

func main() {
	InitRing(5)
}
