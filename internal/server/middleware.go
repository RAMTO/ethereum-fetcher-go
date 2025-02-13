package server

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
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
