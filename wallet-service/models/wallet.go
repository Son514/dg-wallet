package models

import "database/sql"

type Wallet struct {
	balance float64
}

func NewWallet() *Wallet {
	return &Wallet{balance: 0}
}

func (w *Wallet) CreateWallet(database *sql.DB) (int64, error) {
	var id int64
	err := database.QueryRow(
		"INSERT INTO wallet_table DEFAULT VALUES RETURNING wallet_id",
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}