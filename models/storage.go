package storage

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/resonantChaos22/gobank/types"
)

type Storage interface {
	CreateAccount(*types.Account) error
	DeleteAccount(int) error
	UpdateAccount(*types.Account) error
	GetAccountByID(int) (*types.Account, error)
	GetAllAccounts() ([]*types.Account, error)
	GetAccountByNumber(int) (*types.Account, error)
	DeleteAllAccounts() error
	DropAllTables() error
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
	err := store.createAccountTable()
	if err != nil {
		return err
	}
	err = store.createTransactionsTable()
	if err != nil {
		return err
	}
	return err
}

func (store *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
		id serial primary key,
		first_name varchar(50),
		last_name varchar(50),
		password varchar(100),
		number serial,
		balance numeric,
		created_at timestamp
	)`

	_, err := store.db.Exec(query)
	return err
}

func (store *PostgresStore) createTransactionsTable() error {
	query := `CREATE TABLE IF NOT EXISTS transaction (
		id serial primary key,
		sender_id integer not null references account(id),
		receiver_id integer not null references account(id),
		amount numeric not null,
		created_at timestamp
	)`

	_, err := store.db.Query(query)
	return err
}
