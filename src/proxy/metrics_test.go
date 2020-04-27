package main

import (
	"fmt"
	"httptest"
	"log"
	"testing"
)

func TestAcceptMetrics(t *testing.T) {
	InitRing(5)
	httptest.Record()

	for i := 0; i < 5; i++ {
		srcValue := fmt.Sprintf("172.0.20.%d", i)
		go func() {
			cpu := Metrics{
				Data: "1",
				Name: "cpu",
				Src:  srcValue,
			}

			jsonBytes, err := json.Marshal(cpu)
			if err != nil {
				log.Fatal(err)
			}
			req, err := http.NewRequest("POST", "http://127.0.0.1:28002/acceptmetrics", body)
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				log.Fatal(err)
			}

			respRec := httptest.NewRecord()

			AcceptMetrics(respRec, &req)
			if status := respRec.Code; status != http.StatusOK {
				t.Errorf("request %s cpu info error:%v\n", cpu.Src, status)
			}
		}()

		go func() {
			mem := Metrics{
				Data: "10240",
				Name: "memory",
				Src:  srcValue,
			}
			jsonBytes, err := json.Marshal(mem)
			if err != nil {
				log.Fatal(err)
			}
			req, err := http.NewRequest("POST", "http://127.0.0.1:28002/acceptmetrics", body)
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				log.Fatal(err)
			}

			respRec := httptest.NewRecord()

			AcceptMetrics(respRec, &req)
			if status := respRec.Code; status != http.StatusOK {
				t.Errorf("request %s mem info error:%v\n", mem.Src, status)
			}
		}()

		go func() {
			churn := Metrics{
				Data: 10,
				Name: "churn",
				Src:  srcValue,
			}
			jsonBytes, err := json.Marshal(mem)
			if err != nil {
				log.Fatal(err)
			}
			req, err := http.NewRequest("POST", "http://127.0.0.1:28002/acceptmetrics", body)
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				log.Fatal(err)
			}

			respRec := httptest.NewRecord()

			AcceptMetrics(respRec, &req)
			if status := respRec.Code; status != http.StatusOK {
				t.Errorf("request %s churn info error:%v\n", churn.Src, status)
			}
		}()
	}
}
