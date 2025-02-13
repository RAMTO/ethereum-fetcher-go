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

func ValidateTransactionHashes() gin.HandlerFunc {
	return func(c *gin.Context) {
		hashes := c.QueryArray("transactionHashes")

		// Check if empty
		if len(hashes) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "transactionHashes parameter is required",
			})
			return
		}

		for _, hash := range hashes {
			// Check prefix and length first
			if !strings.HasPrefix(hash, "0x") || len(hash) != 66 {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid transaction hash: %s", hash),
				})
				return
			}

			// Try to decode the hex part
			if _, err := hex.DecodeString(hash[2:]); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid transaction hash: %s", hash),
				})
				return
			}

			h := common.HexToHash(hash)

			// If the hash is invalid, it will be zero
			if h == (common.Hash{}) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid transaction hash: %s", hash),
				})
				return
			}
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

		// fmt.Println("hashes", hashes)

		// Check if empty
		if len(hashes) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "transactionHashes parameter is required",
			})
			return
		}

		for _, hash := range hashes {
			// Check prefix and length first
			if !strings.HasPrefix(hash, "0x") || len(hash) != 66 {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid transaction hash: %s", hash),
				})
				return
			}

			// Try to decode the hex part
			if _, err := hex.DecodeString(hash[2:]); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid transaction hash: %s", hash),
				})
				return
			}

			h := common.HexToHash(hash)

			// If the hash is invalid, it will be zero
			if h == (common.Hash{}) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid transaction hash: %s", hash),
				})
				return
			}
		}

		// Store validated hashes in context for handler
		c.Set("validatedHashes", hashes)
		c.Next()
	}
}
