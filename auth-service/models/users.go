package models

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var ErrEmailExists = errors.New("email already exists")

type Users struct {
	email    string
	password string
}

func NewUsers(email, password string) *Users {
	return &Users{
		email:    email,
		password: password,
	}
}

func (u *Users) CreateUsers(database *sql.DB) (int64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	var id int64
	err = database.QueryRow(
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id",
		u.email,
		string(hash),
	).Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, ErrEmailExists
		}
		return 0, err
	}

	return id, nil
}
