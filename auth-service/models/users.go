package models

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

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

func (u *Users) CreateUser(database *sql.DB) (int64, error) {
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

func (u *Users) LoginUser(database *sql.DB) (int64, error) {
	var id int64
	var hash string
	err := database.QueryRow(
		"SELECT id, password FROM users WHERE email = $1",
		u.email,
	).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidCredentials
	}
	if err != nil {
		return 0, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(u.password)); err != nil {
		return 0, ErrInvalidCredentials
	}

	return id, nil
}
