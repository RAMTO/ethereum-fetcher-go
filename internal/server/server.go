package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"ethereum-fetcher-go/internal/database"
	"ethereum-fetcher-go/internal/repository"
)

type Server struct {
	port int

	db              database.Service
	transactionRepo repository.TransactionRepository
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("API_PORT"))
	db := database.New()

	NewServer := &Server{
		port:            port,
		db:              db,
		transactionRepo: repository.NewTransactionRepository(db.DB()),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
