package server

import (
	"ethereum-fetcher-go/internal/models"
	"net/http"
	"os"
	"time"

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
	// Get pre-validated hashes from context
	hashes, exists := c.Get("validatedHashes")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "validated hashes not found in context",
		})
		return
	}

	transactionHashes, ok := hashes.([]string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid type for validated hashes",
		})
		return
	}

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

		userId := int(claims["sub"].(float64))

		foundUser, err := s.store.userRepo.GetByID(c, userId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if foundUser != nil {
			// Check for user transaction association
			for _, hashToCheck := range transactionHashes {
				existingUserTransaction, _ := s.store.userTransactionRepo.GetByTransactionHashAndUserId(c, hashToCheck, foundUser.ID)

				if existingUserTransaction == nil {
					_, err := s.store.userTransactionRepo.Create(c, foundUser.ID, hashToCheck)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						continue
					}
				}
			}
		}
	}

	existingTransactions, _ := s.store.transactionRepo.GetByHashes(c, transactionHashes)

	// If all transactions exist, return them
	if len(existingTransactions) == len(transactionHashes) {
		c.IndentedJSON(http.StatusOK, existingTransactions)
		return
	}

	// Create a map of existing transactions for quick lookup
	existingTxMap := make(map[string]bool)
	for _, tx := range existingTransactions {
		existingTxMap[tx.TransactionHash] = true
	}

	// Fetch new transactions from the network
	newTransactions, err := fetchTransactionsFromNetwork(c, transactionHashes, existingTxMap, s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	allTransactions := append(existingTransactions, newTransactions...)

	c.IndentedJSON(http.StatusOK, allTransactions)
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
	if _, err := s.store.userRepo.Create(c, &user); err != nil {
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

	transactionHashes := make([]string, len(userTransactionIds))
	for i, userTransaction := range userTransactionIds {
		transactionHashes[i] = userTransaction.TransactionHash
	}

	transactions, err := s.store.transactionRepo.GetByHashes(c, transactionHashes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}
