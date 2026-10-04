package main

// TODO: US-3 — Create a Wallet

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"son514/auth-service/database/db"
	"son514/auth-service/jwt"
	"son514/auth-service/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type registerUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerUser(database *sql.DB, c *gin.Context) {
	var request registerUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.NewUsers(request.Email, request.Password)
	id, err := user.CreateUser(database)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, bcrypt.ErrPasswordTooLong):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "email": request.Email})
}

type loginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func loginUser(database *sql.DB, c *gin.Context) {
	var request loginUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.NewUsers(request.Email, request.Password)
	id, err := user.LoginUser(database)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	token, err := jwt.Generate(id, request.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func main() {
	database, err := db.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	router := gin.Default()

	router.POST("/users", func(c *gin.Context) {
		registerUser(database, c)
	})
	router.POST("/login", func(c *gin.Context) {
		loginUser(database, c)
	})

	router.Run()
}
