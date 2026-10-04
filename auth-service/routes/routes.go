package routes

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"son514/auth-service/jwt"
	"son514/auth-service/models"
)

type registerUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Setup(router *gin.Engine, database *sql.DB) {
	router.POST("/users", func(c *gin.Context) {
		registerUser(database, c)
	})
	router.POST("/login", func(c *gin.Context) {
		loginUser(database, c)
	})
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
