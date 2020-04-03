package proxy

import (
	"container/ring"
	"encoding/json"
	"log"
	"os"
)

type DailyLog struct {
	Cid string `json:"cid"`
	Freq int64	`json:"freq"`
	Timestamp string  `json:"timestamp"`
}

const (
	buffer_size = 1024
)

var count = 0

var r *ring.Ring

func Init() {
	r = ring.New(buffer_size)
}

func InsertLog(item *DailyLog) {
	if count < buffer_size {
		r.Value = item
		r = r.Next()
		count++
	} else {
		Persistenize()
		r.Value = item
		count++
	}
}

func ClearLog() {
	r.Do(func(p interface{}) {
		p = nil
	})
}

func Persistenize() {
	f, err := os.OpenFile("log.json",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	r.Do(func(p interface{}) {
		item := *p.(*DailyLog)
		b, err := json.Marshal(item)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write(b); err != nil {
			log.Fatal(err)
		}
	})
}