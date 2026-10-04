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

type ValidatedClaims struct {
	UserID string
	Email  string
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

func Validate(token string) (*ValidatedClaims, error) {
	parsed, err := jwtlib.ParseWithClaims(
		token,
		&claims{},
		func(t *jwtlib.Token) (any, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		},
	)
	if err != nil {
		return nil, err
	}

	parsedClaims, ok := parsed.Claims.(*claims)
	if !ok {
		return nil, jwtlib.ErrTokenMalformed
	}

	return &ValidatedClaims{
		UserID: parsedClaims.Subject,
		Email:  parsedClaims.Email,
	}, nil
}
