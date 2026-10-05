package models

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

var ErrWalletExists = errors.New("wallet already exists")

type Wallet struct {
	userID  int64
	balance string
}

func NewWallet(userID int64) *Wallet {
	return &Wallet{userID: userID}
}

func (w *Wallet) CreateWallet(database *sql.DB) (int64, string, error) {
	var id int64
	var balance string
	err := database.QueryRow(
		"INSERT INTO wallet_table (user_id) VALUES ($1) RETURNING wallet_id, balance",
		w.userID,
	).Scan(&id, &balance)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, "", ErrWalletExists
		}
		return 0, "", err
	}
	w.balance = balance

	return id, balance, nil
}

func (w *Wallet) TopUpWallet(database *sql.DB, walletID int64, amount string) (int64, string, error) {
	var id int64
	var balance string
	err := database.QueryRow(
		"UPDATE wallet_table SET balance = balance + $1::numeric WHERE wallet_id = $2 AND user_id = $3 RETURNING wallet_id, balance",
		amount,
		walletID,
		w.userID,
	).Scan(&id, &balance)
	if err != nil {
		return 0, "", err
	}
	w.balance = balance

	return id, balance, nil
}
