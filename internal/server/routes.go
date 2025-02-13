package server

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/health", s.healthHandler)
	r.GET("/lime/all", s.getAllTransactionsHandler)
	r.GET("/lime/eth", ValidateTransactionHashes(), s.fetchTransactionsHandler)
	r.GET("/lime/eth/:rlphex", ValidateRlpHex(), s.fetchTransactionsHandler)
	r.POST("/lime/register", s.registerUserHandler)
	r.POST("/lime/authenticate", s.authenticateUserHandler)
	r.GET("/lime/my", s.myUserHandler)

	return r
}
