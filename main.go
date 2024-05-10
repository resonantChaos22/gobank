package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("=================Initializing GoBank API====================")
	store, err := NewPostgresStore()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = store.Init()
	if err != nil {
		log.Fatal(err.Error())
	}

	// fmt.Printf("%+v\n", store)

	server := NewServer(":8080", store)
	server.Run()
}
