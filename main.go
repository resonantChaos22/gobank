package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/resonantChaos22/gobank/api"
	storage "github.com/resonantChaos22/gobank/models"
	"github.com/resonantChaos22/gobank/types"
)

func seedAccount(wg *sync.WaitGroup, store storage.Storage, f, l, pw string, accountChan chan *types.Account) {
	defer wg.Done()
	account, err := types.NewAccount(f, l, pw)
	if err != nil {
		log.Fatal(err)
	}

	if err := store.CreateAccount(account); err != nil {
		log.Fatal(err)
	}

	accountChan <- account
}

func seedAccounts(store storage.Storage) {
	var wg sync.WaitGroup
	accountChan := make(chan *types.Account)
	wg.Add(4)
	go seedAccount(&wg, store, "Shreyash", "Pandey", "Test@123", accountChan)
	go seedAccount(&wg, store, "Jeewan", "Singh", "Test@123", accountChan)
	go seedAccount(&wg, store, "Anamay", "Pathak", "Test@123", accountChan)
	go seedAccount(&wg, store, "John", "Doe", "Test@123", accountChan)

	go func() {
		wg.Wait()
		close(accountChan)
	}()

	var accounts []*types.Account
	for account := range accountChan {
		accounts = append(accounts, account)
	}

	err := writeToJSON("accounts.json", accounts)
	if err != nil {
		log.Fatal(err)
	}
}

func writeToJSON(filename string, accounts []*types.Account) error {
	err := os.Remove(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	jsonData, err := json.MarshalIndent(accounts, "", "	")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return err
	}

	log.Printf("Data has been written to %s\n", filename)
	return nil
}

func main() {
	fmt.Println("=================Initializing GoBank API====================")

	seed := flag.Bool("seed", false, "Seed the DB")
	purge := flag.Bool("purge", false, "Purge the DB")
	exit := flag.Bool("exit", false, "Exit the Application")
	flag.Parse()

	store, err := storage.NewPostgresStore()
	if err != nil {
		log.Fatal(err.Error())
	}

	if *purge {
		log.Println("Dropping all tables from DB")
		store.DropAllTables()
	}

	err = store.Init()
	if err != nil {
		log.Fatal(err.Error())
	}

	if *seed {
		log.Println("Seeding Accounts into DB")
		seedAccounts(store)
	}

	if *exit {
		return
	}

	// fmt.Printf("%+v\n", store)

	server := api.NewServer(":8080", store)
	server.Run()
}
