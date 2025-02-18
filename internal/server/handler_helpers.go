package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"ethereum-fetcher-go/internal/contracts"
	"ethereum-fetcher-go/internal/models"
	"fmt"
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

// TxResponse represents a transaction response
type TxResponse struct {
	TxHash   string `json:"txHash"`
	TxStatus string `json:"txStatus"`
}

// getClient creates and returns an Ethereum client
func getClient() (*ethclient.Client, error) {
	ethNodeUrl := os.Getenv("ETH_NODE_URL")
	if ethNodeUrl == "" {
		return nil, fmt.Errorf("ETH_NODE_URL environment variable not set")
	}

	client, err := ethclient.Dial(ethNodeUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	return client, nil
}

// handleError is a helper function to consistently handle errors
func handleError(c *gin.Context, status int, err error, message string) {
	log.Printf("Error: %s: %v", message, err)
	c.JSON(status, gin.H{"error": message})
}

// fetchTransactionsFromNetwork fetches transaction details from the Ethereum network
func fetchTransactionsFromNetwork(c *gin.Context, transactionHashes []string, existingTransactions map[string]bool, s *Server) ([]*models.Transaction, error) {
	client, err := getClient()
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to initialize Ethereum client")
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	var newTransactions []*models.Transaction

	for _, hash := range transactionHashes {
		if existingTransactions[hash] {
			continue
		}

		txHash := common.HexToHash(hash)
		tx, _, err := client.TransactionByHash(c, txHash)
		if err != nil {
			log.Printf("Warning: failed to fetch transaction %s: %v", hash, err)
			continue
		}

		receipt, err := client.TransactionReceipt(c, txHash)
		if err != nil {
			log.Printf("Warning: failed to fetch receipt for %s: %v", hash, err)
			continue
		}

		from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		if err != nil {
			log.Printf("Warning: failed to get sender for %s: %v", hash, err)
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
			log.Printf("Warning: failed to save transaction %s: %v", hash, err)
			continue
		}

		newTransactions = append(newTransactions, transaction)
	}

	return newTransactions, nil
}

// savePersonToContract saves person information to the smart contract and returns transaction details
func savePersonToContract(c *gin.Context, personData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}) (*TxResponse, error) {
	client, err := getClient()
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to initialize Ethereum client")
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	address := common.HexToAddress(os.Getenv("CONTRACT_ADDRESS"))
	if address == common.HexToAddress("0x0") {
		handleError(c, http.StatusInternalServerError, fmt.Errorf("invalid contract address"), "Invalid contract address")
		return nil, fmt.Errorf("invalid contract address")
	}

	instance, err := contracts.NewContracts(address, client)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to initialize contract")
		return nil, fmt.Errorf("failed to initialize contract: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Invalid private key")
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		handleError(c, http.StatusInternalServerError, fmt.Errorf("invalid public key"), "Failed to process public key")
		return nil, fmt.Errorf("failed to process public key")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to get nonce")
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to get gas price")
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	increasedGasPrice := new(big.Int).Mul(gasPrice, big.NewInt(130))
	increasedGasPrice = increasedGasPrice.Div(increasedGasPrice, big.NewInt(100))

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to get chain ID")
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to create transaction authenticator")
		return nil, fmt.Errorf("failed to create transaction authenticator: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(500000)
	auth.GasPrice = increasedGasPrice

	txHash, err := instance.SetPersonInfo(auth, personData.Name, big.NewInt(int64(personData.Age)))
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to set person information")
		return nil, fmt.Errorf("failed to set person information: %w", err)
	}

	receipt, err := bind.WaitMined(context.Background(), client, txHash)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err, "Failed to confirm transaction")
		return nil, fmt.Errorf("failed to confirm transaction: %w", err)
	}

	txStatus := "success"
	if receipt.Status == 0 {
		txStatus = "failed"
	}

	return &TxResponse{
		TxHash:   txHash.Hash().Hex(),
		TxStatus: txStatus,
	}, nil
}
