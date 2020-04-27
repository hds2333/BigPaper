package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestAcceptMetrics(t *testing.T) {
	log.Println("hello world")
	InitRing()

	wg := sync.WaitGroup{}
	wg.Add(15)
	for i := 0; i < 5; i++ {
		srcValue := fmt.Sprintf("172.0.20.%d", i)
		go func() {
			cpu := Metric{
				Data: 1.0,
				Name: "cpu",
				Src:  srcValue,
			}

			defer wg.Done()

			jsonBytes, err := json.Marshal(cpu)
			if err != nil {
				log.Fatal(err)
			}
			body := bytes.NewReader(jsonBytes)
			req, err := http.NewRequest("POST", "http://127.0.0.1:28002/acceptmetrics", body)
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				log.Fatal(err)
			}

			respRec := httptest.NewRecorder()

			AcceptMetrics(respRec, req)
			if status := respRec.Code; status != http.StatusOK {
				t.Errorf("request %s cpu info error:%v\n", cpu.Src, status)
			}
		}()

		go func() {
			mem := Metric{
				Data: 10240.0,
				Name: "memory",
				Src:  srcValue,
			}
			defer wg.Done()
			jsonBytes, err := json.Marshal(mem)
			if err != nil {
				log.Fatal(err)
			}
			body := bytes.NewReader(jsonBytes)
			req, err := http.NewRequest("POST", "http://127.0.0.1:28002/acceptmetrics", body)
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				log.Fatal(err)
			}

			respRec := httptest.NewRecorder()

			AcceptMetrics(respRec, req)
			if status := respRec.Code; status != http.StatusOK {
				t.Errorf("request %s mem info error:%v\n", mem.Src, status)
			}
		}()

		go func() {
			churn := Metric{
				Data: 10.0,
				Name: "churn",
				Src:  srcValue,
			}
			defer wg.Done()
			jsonBytes, err := json.Marshal(churn)
			if err != nil {
				log.Fatal(err)
			}
			body := bytes.NewReader(jsonBytes)
			req, err := http.NewRequest("POST", "http://127.0.0.1:28002/acceptmetrics", body)
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				log.Fatal(err)
			}

			respRec := httptest.NewRecorder()

			AcceptMetrics(respRec, req)
			if status := respRec.Code; status != http.StatusOK {
				t.Errorf("request %s churn info error:%v\n", churn.Src, status)
			}
		}()
	}

	if cpuBuffer == nil || memBuffer == nil || churnBuffer == nil {
		log.Fatal("Ring Buffer not initilized")
	}

	wg.Wait()
	CalcGrade()
}
