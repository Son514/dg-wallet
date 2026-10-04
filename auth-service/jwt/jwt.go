package jwt

import (
	"os"
	"strconv"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type claims struct {
	Email string `json:"email"`
	jwtlib.RegisteredClaims
}

func Generate(id int64, email string) (string, error) {
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims{
		Email: email,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   strconv.FormatInt(id, 10),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
