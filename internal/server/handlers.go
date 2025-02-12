package server

import (
	"encoding/hex"
	"ethereum-fetcher-go/internal/models"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

func (s *Server) getAllTransactionsHandler(c *gin.Context) {
	txs, err := s.store.transactionRepo.GetAll(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

func (s *Server) fetchTransactionsHandler(c *gin.Context) {
	param := c.Query("transactionHashes")

	var user *models.User

	// Get the token from the request
	tokenString := c.GetHeader("Authorization")
	if tokenString != "" {
		// Validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		username := claims["username"].(string)

		foundUser, err := s.store.userRepo.GetByUsername(c, username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		user = foundUser
	}

	fmt.Println(user)

	// Validate transaction hash
	if param == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transactionHashes parameter is required",
		})
		return
	}

	existingTransaction, _ := s.store.transactionRepo.GetByHash(c, param)

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

	receipt, err := client.TransactionReceipt(c, txHash)
	if err != nil {
		log.Fatal(err)
	}

	from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		log.Fatal(err)
	}

	transaction := &models.Transaction{
		TransactionHash:   tx.Hash().Hex(),
		TransactionStatus: int(receipt.Status),
		To:                tx.To().Hex(),
		From:              from.Hex(),
		ContractAddress:   tx.To().Hex(),
		LogsCount:         len(receipt.Logs),
		Value:             int(tx.Value().Int64()),
		BlockHash:         receipt.BlockHash.Hex(),
		BlockNumber:       int(receipt.BlockNumber.Int64()),
		Input:             hex.EncodeToString(tx.Data()),
	}

	if err := s.store.transactionRepo.Create(c, transaction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if user != nil {
		// Check for user transaction association
		existingUserTransaction, _ := s.store.userTransactionRepo.GetByTransactionHashAndUserId(c, transaction.TransactionHash, user.ID)

		fmt.Println("Existing user transaction", existingUserTransaction)

		if existingUserTransaction == nil {
			if err := s.store.userTransactionRepo.Create(c, user.ID, transaction.TransactionHash); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
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
	existingUser, _ := s.store.userRepo.GetByUsername(c, user.Username)
	if existingUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already exists"})
		return
	}

	// Create user
	if err := s.store.userRepo.Create(c, &user); err != nil {
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

	existingUser, _ := s.store.userRepo.GetByUsername(c, user.Username)
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
		"sub":      existingUser.ID,
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

func (s *Server) myUserHandler(c *gin.Context) {
	// Get the token from the request
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided"})
		return
	}

	// Validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Get the username from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	userId := claims["sub"].(float64)

	userTransactionIds, err := s.store.userTransactionRepo.GetTransactionsByUserId(c, int(userId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("User transaction IDs", userTransactionIds)

	c.JSON(http.StatusOK, userTransactionIds)
}
