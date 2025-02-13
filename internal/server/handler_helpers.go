package server

import (
	"encoding/hex"
	"encoding/json"
	"ethereum-fetcher-go/internal/models"
	"log"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

// ResponseHelper wraps common JSON response functionality
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// ErrorResponse represents a standard error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondWithError is a helper to send error responses
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}

// validateRequest is a helper to validate and decode JSON requests
func validateRequest(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(&dst); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return false
	}
	return true
}

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
