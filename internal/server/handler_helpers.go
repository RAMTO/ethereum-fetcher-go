package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"ethereum-fetcher-go/internal/contracts"
	"ethereum-fetcher-go/internal/models"
	"log"
	"math/big"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

// Fetch transactions from the network
func fetchTransactionsFromNetwork(c *gin.Context, transactionHashes []string, existingTransactions map[string]bool, s *Server) ([]*models.Transaction, error) {
	ethNodeUrl := os.Getenv("ETH_NODE_URL")

	// Dial the Ethereum node
	client, err := ethclient.Dial(ethNodeUrl)
	if err != nil {
		return nil, err
	}

	var newTransactions []*models.Transaction

	for _, hash := range transactionHashes {
		// If the transaction already exists, skip it
		if existingTransactions[hash] {
			continue
		}

		txHash := common.HexToHash(hash)

		tx, _, err := client.TransactionByHash(c, txHash)
		if err != nil {
			log.Printf("Error fetching transaction %s: %v", hash, err)
			continue
		}

		receipt, err := client.TransactionReceipt(c, txHash)
		if err != nil {
			log.Printf("Error fetching receipt for %s: %v", hash, err)
			continue
		}

		from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		if err != nil {
			log.Printf("Error getting sender for %s: %v", hash, err)
			continue
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

		if _, err := s.store.transactionRepo.Create(c, transaction); err != nil {
			log.Printf("Error saving transaction %s: %v", hash, err)
			continue
		}

		newTransactions = append(newTransactions, transaction)
	}

	return newTransactions, nil
}

func savePersonToContract(c *gin.Context, personData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}) (txResponse struct {
	TxHash   string `json:"txHash"`
	TxStatus string `json:"txStatus"`
}, err error) {
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

	personName := personData.Name
	personAge := big.NewInt(int64(personData.Age))

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

	txResponse = struct {
		TxHash   string `json:"txHash"`
		TxStatus string `json:"txStatus"`
	}{
		TxHash:   txHash.Hash().Hex(),
		TxStatus: txStatus,
	}

	return txResponse, nil
}
