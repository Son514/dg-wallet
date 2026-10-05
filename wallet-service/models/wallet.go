package models

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

var ErrWalletExists = errors.New("wallet already exists")

type Wallet struct {
	userID  int64
	balance float64
}

func NewWallet(userID int64) *Wallet {
	return &Wallet{userID: userID}
}

func (w *Wallet) CreateWallet(database *sql.DB) (int64, error) {
	var id int64
	err := database.QueryRow(
		"INSERT INTO wallet_table (user_id) VALUES ($1) RETURNING wallet_id",
		w.userID,
	).Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, ErrWalletExists
		}
		return 0, err
	}

	return id, nil
}