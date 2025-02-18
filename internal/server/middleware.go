package server

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gin-gonic/gin"
)

func validateHashes(hashes []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("transactionHashes parameter is required")
	}

	for _, hash := range hashes {
		// Check prefix and length first
		if !strings.HasPrefix(hash, "0x") || len(hash) != 66 {
			return fmt.Errorf("invalid transaction hash: %s", hash)
		}

		// Try to decode the hex part
		if _, err := hex.DecodeString(hash[2:]); err != nil {
			return fmt.Errorf("invalid transaction hash: %s", hash)
		}

		h := common.HexToHash(hash)

		// If the hash is invalid, it will be zero
		if h == (common.Hash{}) {
			return fmt.Errorf("invalid transaction hash: %s", hash)
		}
	}

	return nil
}

func ValidateTransactionHashes() gin.HandlerFunc {
	return func(c *gin.Context) {
		hashes := c.QueryArray("transactionHashes")

		if err := validateHashes(hashes); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Store validated hashes in context for handler
		c.Set("validatedHashes", hashes)
		c.Next()
	}
}

func ValidateRlpHex() gin.HandlerFunc {
	return func(c *gin.Context) {
		param := c.Param("rlphex")

		// Remove "0x" prefix if present
		if strings.HasPrefix(param, "0x") {
			param = param[2:]
		}

		rlpData, err := hex.DecodeString(param)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid RLP hex: " + err.Error()})
			return
		}

		// Create a slice to store the raw bytes
		var rawHashes [][]byte

		// Decode the RLP data directly into the slice
		if err := rlp.DecodeBytes(rlpData, &rawHashes); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode RLP data: " + err.Error()})
			return
		}

		// Convert byte slices to hex strings
		hashes := make([]string, len(rawHashes))
		for i, hash := range rawHashes {
			hashes[i] = "0x" + hex.EncodeToString(hash)
		}

		if err := validateHashes(hashes); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Store validated hashes in context for handler
		c.Set("validatedHashes", hashes)
		c.Next()
	}
}

func ValidatePersonData() gin.HandlerFunc {
	return func(c *gin.Context) {
		var personData struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		if err := c.ShouldBindJSON(&personData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if personData.Age < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Age cannot be negative"})
			return
		}

		if personData.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name cannot be empty"})
			return
		}

		c.Set("personData", personData)
		c.Next()
	}
}
