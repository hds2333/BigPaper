package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

/*
func TestAddOp(t *testing.T) {

}
*/
func TestPut(t *testing.T) {
	bodyContent, err := ioutil.ReadFile("./datanode_test.go")
	body := ioutil.NopCloser(bytes.NewReader(bodyContent))

	req, err := http.NewRequest("POST", "http://127.0.0.1:28002/put", body)
	req.Header.Set("Content-Type", "application/octet-stream")
	if err != nil {
		log.Fatal(err)
	}

	respRec := httptest.NewRecorder()
	Put(respRec, req)
	if status := respRec.Code; status != http.StatusOK {
		t.Errorf("http error status:%v", status)
	}
}

func TestSendAddRequest(t *testing.T) {
	dstIP := "127.0.0.1"
	cid := "QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y"
	statusCode := SendAddRequest(dstIP, cid)
	if statusCode != 200 {
		t.Errorf("expected status:%d, actual status:%d\n", 200, statusCode)
	}
}

func TestSendDelRequest(t *testing.T) {
	adjItem := AdjItem{
		Policy: 0,
		Cid:    "QmPaGBsLNXBBkAG1WzgTiv7ry63DsV3nqNPHWtRsPVgcFm",
		Delta:  1,
	}
	dstIP := "127.0.0.1"
	statusCode := SendDelRequest(dstIP, &adjItem)
	if statusCode != 200 {
		t.Errorf("expected status:%d, actual status:%d\n", 200, statusCode)
	}
}

/*
func TestAddReplica(t *testing.T) {

}

func TestDelReplica(t *testing.T) {

}
*/
/*
func TestCalCidByContent(t *testing.T) {
	cmd := exec.Command("ipfs", "add", "./datanode_test.go")
	cmd
	bodyContent, err := ioutil.ReadFile("./datanode_test.go")
	body := ioutil.NopCloser(bytes.NewReader(bodyContent))

	req, err := http.NewRequest("POST", "http://127.0.0.1:28002/put", body)
	req.Header.Add("Content-Type", "binary/octet-stream")
	if err != nil {
		log.Fatal(err)
	}

	respRec := httptest.NewRecorder()
	Put(respRec, req)
	if status := respRec.Code; status != http.StatusOK {
		t.Errorf("http error status:%v", status)
	}
}
*/
