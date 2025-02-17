package server

import (
	"context"
	"crypto/ecdsa"
	"ethereum-fetcher-go/internal/contracts"
	"ethereum-fetcher-go/internal/models"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

func (s *Server) savePersonHandler(c *gin.Context) {
	personData, exists := c.Get("personData")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Person data not found in context"})
		return
	}

	// Create client
	ethNodeUrl := os.Getenv("ETH_NODE_URL")

	// Dial the Ethereum node
	client, err := ethclient.Dial(ethNodeUrl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Client not found"})
		return
	}

	// Get contract instance
	address := common.HexToAddress(os.Getenv("CONTRACT_ADDRESS"))
	instance, err := contracts.NewContracts(address, client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Contract not found"})
		return
	}

	// Get private key
	privateKey, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Private key not found"})
		return
	}

	// Get public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Casting public key to ECDSA"})
		return
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nonce"})
		return
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get gas price"})
		return
	}

	// Increase gas price by 30% to speed up transaction
	increasedGasPrice := new(big.Int).Mul(gasPrice, big.NewInt(130))
	increasedGasPrice = increasedGasPrice.Div(increasedGasPrice, big.NewInt(100))

	// Get chain ID first
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chain ID"})
		return
	}

	// Create auth with chain ID
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create auth"})
		return
	}

	// Set the transaction parameters
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(500000)
	auth.GasPrice = increasedGasPrice

	data := personData.(struct {
		Name string `json:"name" binding:"required"`
		Age  int    `json:"age" binding:"required"`
	})

	personName := data.Name
	personAge := big.NewInt(int64(data.Age))

	txHash, err := instance.SetPersonInfo(auth, personName, personAge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed call to set person info"})
		return
	}

	receipt, err := bind.WaitMined(context.Background(), client, txHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to wait for transaction to be mined"})
		return
	}

	txStatus := "success"
	if receipt.Status == 0 {
		txStatus = "failed"
	}

	txResponse := struct {
		TxHash   string `json:"txHash"`
		TxStatus string `json:"txStatus"`
	}{
		TxHash:   txHash.Hash().Hex(),
		TxStatus: txStatus,
	}

	c.JSON(http.StatusOK, txResponse)
}
