package server

import (
	"ethereum-fetcher-go/internal/models"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/health", s.healthHandler)
	r.GET("/lime/all", s.getAllTransactionsHandler)
	r.GET("/lime/eth", s.fetchTransactionsHandler)
	r.POST("/lime/register", s.registerUserHandler)
	r.POST("/lime/authenticate", s.authenticateUserHandler)

	return r
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

func (s *Server) getAllTransactionsHandler(c *gin.Context) {
	txs, err := s.transactionRepo.GetAll(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

func (s *Server) fetchTransactionsHandler(c *gin.Context) {
	param := c.Query("transactionHashes")

	// Validate transaction hash
	if param == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transactionHashes parameter is required",
		})
		return
	}

	existingTransaction, _ := s.transactionRepo.GetByHash(c, param)

	if existingTransaction != nil {
		c.IndentedJSON(http.StatusOK, existingTransaction)
		return
	}

	ethNodeUrl := os.Getenv("ETH_NODE_URL")

	client, err := ethclient.Dial(ethNodeUrl)
	if err != nil {
		log.Fatal(err)
	}

	txHash := common.HexToHash(param)

	tx, _, err := client.TransactionByHash(c, txHash)
	if err != nil {
		log.Fatal(err)
	}

	transaction := &models.Transaction{
		TransactionHash:   tx.Hash().Hex(),
		TransactionStatus: 1,
		To:                tx.To().Hex(),
		ContractAddress:   tx.To().Hex(),
		LogsCount:         0,
		Value:             int(tx.Value().Int64()),
	}

	if err := s.transactionRepo.Create(c, transaction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, transaction)
}

func (s *Server) registerUserHandler(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, _ := s.userRepo.GetByUsername(c, user.Username)
	if existingUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already exists"})
		return
	}

	// Create user
	if err := s.userRepo.Create(c, &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func (s *Server) authenticateUserHandler(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingUser, _ := s.userRepo.GetByUsername(c, user.Username)
	if existingUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if existingUser.Password != user.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(time.Minute * 15).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
