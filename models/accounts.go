package storage

import (
	"database/sql"
	"fmt"

	"github.com/resonantChaos22/gobank/types"
)

func (store *PostgresStore) CreateAccount(account *types.Account) error {
	query := `
	INSERT INTO account (first_name, last_name, password, number, balance, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := store.db.Query(query, account.FirstName, account.LastName, account.EncryptedPassword, account.Number, account.Balance, account.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (store *PostgresStore) GetAccountByID(id int) (*types.Account, error) {
	query := `SELECT * FROM account WHERE id=$1`
	rows, err := store.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account with id %d not found", id)
}

func (store *PostgresStore) GetAccountByNumber(id int) (*types.Account, error) {
	query := `SELECT * FROM account WHERE number=$1`
	rows, err := store.db.Query(query, int64(id))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account with number %d not found", id)
}

func (store *PostgresStore) GetAllAccounts() ([]*types.Account, error) {
	query := `SELECT * FROM account`
	rows, err := store.db.Query(query)
	if err != nil {
		return nil, err
	}

	accounts := []*types.Account{}
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

func (store *PostgresStore) UpdateAccount(account *types.Account) error {
	return nil
}

func (store *PostgresStore) DeleteAllAccounts() error {
	queries := []string{
		`DELETE * FROM account`,
	}
	for _, query := range queries {
		_, err := store.db.Query(query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (store *PostgresStore) DropAllTables() error {
	queries := []string{
		`DROP TABLE IF EXISTS account CASCADE`,
		`DROP TABLE IF EXISTS transaction CASCADE`,
	}

	for _, query := range queries {
		_, err := store.db.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}

func scanIntoAccount(rows *sql.Rows) (*types.Account, error) {
	account := new(types.Account)

	err := rows.Scan(&account.ID, &account.FirstName, &account.LastName, &account.EncryptedPassword, &account.Number, &account.Balance, &account.CreatedAt)
	if err != nil {
		return nil, err
	}

	return account, nil
}
