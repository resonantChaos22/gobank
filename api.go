package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type APIError struct {
	OriginalError error
	Status        int
}

type APIFunc func(w http.ResponseWriter, r *http.Request) (any, *APIError)

func (e *APIError) Error() string {
	return e.OriginalError.Error()
}

func MakeAPIError(err error, status int) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	} else {
		return &APIError{
			OriginalError: err,
			Status:        status,
		}
	}
}

func MakeHTTPHandler(f APIFunc) http.HandlerFunc {
	timeout := time.Second * 30
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		dataChan := make(chan any, 1)
		errChan := make(chan *APIError, 1)
		go func() {
			data, err := f(w, r)
			if err != nil {
				errChan <- err
				return
			}
			dataChan <- data
		}()

		select {
		case <-ctx.Done():
			WriteJSON(w, ErrorResponse{Error: "request timed out"}, http.StatusRequestTimeout)
		case data := <-dataChan:
			WriteJSON(w, data, http.StatusOK)
		case err := <-errChan:
			WriteJSON(w, ErrorResponse{Error: err.Error()}, err.Status)
		}
	}
}

func WriteJSON(w http.ResponseWriter, data any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func getId(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid id given: %s", idStr)
	}

	return id, nil
}

func NewServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() error {
	router := mux.NewRouter()
	router.HandleFunc("/accounts", MakeHTTPHandler(s.handleAccounts))
	router.HandleFunc("/accounts/{id}", MakeHTTPHandler(s.handleAccountsByID))
	router.HandleFunc("/transfer", MakeHTTPHandler(s.handleTransfer))

	return http.ListenAndServe(s.listenAddr, router)
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
	createAccountReq := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return nil, MakeAPIError(err, http.StatusBadRequest)
	}
	defer r.Body.Close()

	account := NewAccount(createAccountReq.FirstName, createAccountReq.LastName)
	if err := s.store.CreateAccount(account); err != nil {
		return nil, MakeAPIError(err, http.StatusInternalServerError)
	}
	return account, nil
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

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) (any, *APIError) {
	if r.Method == "POST" {

		transferReq := new(TransferRequest)
		if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
			return nil, MakeAPIError(err, http.StatusBadRequest)
		}
		defer r.Body.Close()
		fmt.Printf("%+v\n", transferReq)
		return transferReq, nil
	}

	return nil, MakeAPIError(fmt.Errorf("method not allowed - %s", r.Method), http.StatusBadRequest)
}
