package main

import (
	"io/ioutil"
	"log"
)

func main() {
	message := []byte("hello, gophers")
	err := ioutil.WriteFile("test", message, 0644)
	if err != nil {
		log.Fatal(err)
	}
	cont, err := ioutil.ReadFile("test")
	if err == nil {
		log.Println(cont)
	}
}
