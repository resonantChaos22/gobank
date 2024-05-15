package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Number   int64  `json:"number"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Number int64  `json:"number"`
	Token  string `json:"token"`
}

type SignupResponse struct {
	Account *Account `json:"account"`
	Token   string   `json:"token"`
}

type CreateAccountRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type TransferRequest struct {
	ToAccount int `json:"to"`
	Value     int `json:"value"`
}

// !	`json:"-"` makes it so that the value is not present in the JSON response
type Account struct {
	ID                int       `json:"id"`
	FirstName         string    `json:"firstName"`
	LastName          string    `json:"lastName"`
	Number            int64     `json:"number"`
	EncryptedPassword string    `json:"-"`
	Balance           int64     `json:"balance"`
	CreatedAt         time.Time `json:"createdAt"`
}

func (acc *Account) ValidatePassword(passwd string) error {
	err := bcrypt.CompareHashAndPassword([]byte(acc.EncryptedPassword), []byte(passwd))
	if err != nil {
		return err
	}

	return nil
}

func NewAccount(FirstName, LastName, Password string) (*Account, error) {
	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:                rand.Intn(1000),
		FirstName:         FirstName,
		LastName:          LastName,
		Number:            int64(rand.Intn(1000000)),
		EncryptedPassword: string(encryptedPassword),
		Balance:           0,
		CreatedAt:         time.Now().UTC(),
	}, nil
}
