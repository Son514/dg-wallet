package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"son514/auth-service/database/db"
	"son514/auth-service/models"
)

type createUsersRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func createUsers(database *sql.DB, c *gin.Context) {
	var request createUsersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.NewUsers(request.Email, request.Password)
	id, err := user.CreateUsers(database)
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

func main() {
	database, err := db.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	router := gin.Default()

	router.POST("/users", func(c *gin.Context) {
		createUsers(database, c)
	})
	router.Run()
}
