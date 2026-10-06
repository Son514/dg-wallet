package models

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

var (
	ErrWalletExists        = errors.New("wallet already exists")
	ErrNotWalletOwner      = errors.New("you do not own this wallet")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

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
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", w.ownershipError(database, walletID)
		}
		return 0, "", err
	}
	w.balance = balance

	return id, balance, nil
}

func (w *Wallet) CheckBalance(database *sql.DB, walletID int64) (int64, string, error) {
	var id int64
	var balance string
	err := database.QueryRow(
		"SELECT wallet_id, balance FROM wallet_table WHERE wallet_id = $1 AND user_id = $2",
		walletID,
		w.userID,
	).Scan(&id, &balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", w.ownershipError(database, walletID)
		}
		return 0, "", err
	}
	w.balance = balance

	return id, balance, nil
}

func (w *Wallet) TransferMoney(database *sql.DB, fromWalletID int64, toWalletID int64, amount string) (int64, string, error) {
	var id int64
	var balance string
	err := database.QueryRow(
		"UPDATE wallet_table SET balance = balance - $1::numeric WHERE wallet_id = $2 AND user_id = $3 AND balance >= $1::numeric RETURNING wallet_id, balance",
		amount,
		fromWalletID,
		w.userID,
	).Scan(&id, &balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var ownerID int64
			if err2 := database.QueryRow(
				"SELECT user_id FROM wallet_table WHERE wallet_id = $1",
				fromWalletID,
			).Scan(&ownerID); err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					return 0, "", sql.ErrNoRows
				}
				return 0, "", err2
			}
			if ownerID != w.userID {
				return 0, "", ErrNotWalletOwner
			}
			return 0, "", ErrInsufficientBalance
		}
		return 0, "", err
	}
	_, err = database.Exec(
		"UPDATE wallet_table SET balance = balance + $1::numeric WHERE wallet_id = $2",
		amount,
		toWalletID,
	)
	if err != nil {
		return 0, "", err
	}
	w.balance = balance

	return id, balance, nil
}

// ownershipError distinguishes a missing wallet from someone else's. It runs
// only after an ownership-scoped query already returned sql.ErrNoRows, so any
// row it finds here cannot belong to the caller.
func (w *Wallet) ownershipError(database *sql.DB, walletID int64) error {
	var ownerID int64
	if err := database.QueryRow(
		"SELECT user_id FROM wallet_table WHERE wallet_id = $1",
		walletID,
	).Scan(&ownerID); err != nil {
		return err
	}
	return ErrNotWalletOwner
}
