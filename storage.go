package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccountByID(int) (*Account, error)
	GetAllAccounts() ([]*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=shreyash dbname=gobank password=eatsleepcode sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (store *PostgresStore) Init() error {
	return store.createAccountTable()
}

func (store *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
		id serial primary key,
		first_name varchar(50),
		last_name varchar(50),
		number serial,
		balance numeric,
		created_at timestamp
	)`

	_, err := store.db.Exec(query)
	return err
}

func (store *PostgresStore) CreateAccount(account *Account) error {
	query := `
	INSERT INTO account (first_name, last_name, number, balance, created_at)
	VALUES ($1, $2, $3, $4, $5)
	`
	resp, err := store.db.Query(query, account.FirstName, account.LastName, account.Number, account.Balance, account.CreatedAt)
	if err != nil {
		return err
	}
	log.Printf("%+v\n", resp)
	return nil
}

func (store *PostgresStore) GetAccountByID(id int) (*Account, error) {
	query := `SELECT * FROM account WHERE id=$1`
	rows, err := store.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, MakeAPIError(fmt.Errorf("account with id %d not found", id), http.StatusNotFound)
}

func (store *PostgresStore) GetAllAccounts() ([]*Account, error) {
	query := `SELECT * FROM account`
	rows, err := store.db.Query(query)
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}
	for rows.Next() {
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (store *PostgresStore) DeleteAccount(id int) error {
	if _, err := store.GetAccountByID(id); err != nil {
		return err
	}

	query := `DELETE FROM account WHERE id=$1`
	_, err := store.db.Query(query, id)
	return err
}

func (store *PostgresStore) UpdateAccount(account *Account) error {
	return nil
}

func scanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := new(Account)

	err := rows.Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreatedAt)
	if err != nil {
		return nil, err
	}

	return account, nil
}
