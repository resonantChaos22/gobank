package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func InvokeInvalidError(w http.ResponseWriter) {
	WriteJSON(w, ErrorResponse{Error: "invalid token"}, http.StatusForbidden)
}

// Basically a middleware
func withJWTAuth(handler http.HandlerFunc, store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling JWT Auth")

		tokenString := r.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)

		if err != nil {
			log.Println(err.Error())
			return
		}

		if !token.Valid {
			InvokeInvalidError(w)
			return
		}

		//	This can only work for jwt.MapClaims as we are type asserting of the interface Claims
		//	TODO: Use your own claims
		claims := token.Claims.(jwt.MapClaims)

		userId, err := getId(r)
		if err != nil {
			InvokeInvalidError(w)
			return
		}

		account, err := store.GetAccountByID(userId)
		if err != nil {
			InvokeInvalidError(w)
			return
		}

		accNum := int64(claims["accountNumber"].(float64))

		if account.Number != accNum {
			InvokeInvalidError(w)
			return
		}

		handler(w, r)
	}
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     15000,
		"accountNumber": account.Number,
	}

	//	TODO:	Use Environment Variable
	secret := "eatsleepcode"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := "eatsleepcode"
	//	TODO:	Implement environment Variables
	//	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, MakeAPIError(fmt.Errorf("unexpected signing method: %v", token.Header["alg"]), http.StatusMethodNotAllowed)
		}

		return []byte(secret), nil
	})
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
	router.HandleFunc("/accounts/{id}", withJWTAuth(MakeHTTPHandler(s.handleAccountsByID), s.store))
	router.HandleFunc("/transfer", MakeHTTPHandler(s.handleTransfer))
	router.HandleFunc("/login", MakeHTTPHandler(s.handleLogin))

	return http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) (any, *APIError) {
	if r.Method == "POST" {

		var req LoginRequest

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

		return &LoginResponse{
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
	createAccountReq := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return nil, MakeAPIError(err, http.StatusBadRequest)
	}
	defer r.Body.Close()

	account, err := NewAccount(createAccountReq.FirstName, createAccountReq.LastName, createAccountReq.Password)
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

	return &SignupResponse{
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
