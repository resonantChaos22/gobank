package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/resonantChaos22/gobank/types"
)

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) (any, *APIError) {
	if r.Method == "POST" {

		var req types.LoginRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, MakeAPIError(err, http.StatusBadRequest)
		}

		acc, err := s.store.GetAccountByNumber(int(req.Number))
		if err != nil {
			return nil, MakeAPIError(err, http.StatusBadRequest)
		}

		err = acc.ValidatePassword(req.Password)
		if err != nil {
			return nil, MakeAPIError(fmt.Errorf("wrong username or password"), http.StatusForbidden)
		}

		token, err := createJWT(acc)
		if err != nil {
			return nil, MakeAPIError(err, http.StatusInternalServerError)
		}

		return &types.LoginResponse{
			Number: acc.Number,
			Token:  token,
		}, nil
	}

	return nil, MakeAPIError(fmt.Errorf("method not allowed - %s", r.Method), http.StatusBadRequest)

}

func (s *APIServer) handleAccounts(w http.ResponseWriter, r *http.Request) (any, *APIError) {
	if r.Method == "GET" {
		return s.handleGetAllAccounts(w, r)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}

	return nil, MakeAPIError(fmt.Errorf("method not allowed - %s", r.Method), http.StatusBadRequest)
}

func (s *APIServer) handleAccountsByID(w http.ResponseWriter, r *http.Request) (any, *APIError) {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}
	return nil, MakeAPIError(fmt.Errorf("method not allowed - %s", r.Method), http.StatusBadRequest)
}

func (s *APIServer) handleGetAllAccounts(_ http.ResponseWriter, _ *http.Request) (any, *APIError) {
	accounts, err := s.store.GetAllAccounts()
	if err != nil {
		return nil, MakeAPIError(err, http.StatusBadRequest)
	}

	return accounts, nil
}

func (s *APIServer) handleCreateAccount(_ http.ResponseWriter, r *http.Request) (any, *APIError) {
	createAccountReq := new(types.CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return nil, MakeAPIError(err, http.StatusBadRequest)
	}
	defer r.Body.Close()

	account, err := types.NewAccount(createAccountReq.FirstName, createAccountReq.LastName, createAccountReq.Password)
	if err != nil {
		return nil, MakeAPIError(err, http.StatusInternalServerError)
	}
	if err := s.store.CreateAccount(account); err != nil {
		return nil, MakeAPIError(err, http.StatusInternalServerError)
	}

	tokenString, err := createJWT(account)
	if err != nil {
		return nil, MakeAPIError(err, http.StatusBadRequest)
	}

	return &types.SignupResponse{
		Account: account,
		Token:   tokenString,
	}, nil
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) (any, *APIError) {
	id, err := getId(r)
	if err != nil {
		return nil, MakeAPIError(err, http.StatusBadRequest)
	}

	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return nil, &APIError{
			OriginalError: err,
			Status:        http.StatusNotFound,
		}
	}
	return account, nil
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) (any, *APIError) {
	id, err := getId(r)
	if err != nil {
		return nil, MakeAPIError(err, http.StatusBadRequest)
	}
	err = s.store.DeleteAccount(id)
	if err != nil {
		return nil, MakeAPIError(err, http.StatusInternalServerError)
	}
	return map[string]string{"message": fmt.Sprintf("deleted account with id %d", id)}, nil
}
