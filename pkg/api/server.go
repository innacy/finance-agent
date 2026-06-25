package api

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/db"
)

type OverviewResponse struct {
	TotalBalance   float64 `json:"total_balance"`
	TotalAccounts  int     `json:"total_accounts"`
	TotalDebit     float64 `json:"total_debit"`
	TotalCredit    float64 `json:"total_credit"`
	TransactionCount int   `json:"transaction_count"`
}

type CategorySpend struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Count    int     `json:"count"`
}

type Server struct {
	db     *db.Client
	userID string
	router *gin.Engine
}

func NewServer(dbClient *db.Client, userID string) *Server {
	gin.SetMode(gin.TestMode)

	s := &Server{
		db:     dbClient,
		userID: userID,
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.Default())

	api := router.Group("/api")
	{
		api.GET("/accounts", s.getAccounts)
		api.GET("/transactions", s.getTransactions)
		api.GET("/overview", s.getOverview)
		api.GET("/categories", s.getCategories)
		api.GET("/brain/status", s.getBrainStatus)
		api.GET("/spend/categories", s.getSpendByCategory)
	}

	s.router = router
	return s
}

func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) ServeStatic(distPath string) {
	absPath, _ := filepath.Abs(distPath)
	staticFS := os.DirFS(absPath)

	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:]
		}

		f, err := fs.Stat(staticFS, path)
		if err == nil && !f.IsDir() {
			c.FileFromFS(c.Request.URL.Path, http.FS(staticFS))
			return
		}

		c.FileFromFS("index.html", http.FS(staticFS))
	})
}

func (s *Server) ServeEmbedded(embedded fs.FS) {
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:]
		}

		f, err := fs.Stat(embedded, path)
		if err == nil && !f.IsDir() {
			c.FileFromFS(c.Request.URL.Path, http.FS(embedded))
			return
		}

		c.FileFromFS("index.html", http.FS(embedded))
	})
}

func (s *Server) getAccounts(c *gin.Context) {
	accounts, err := s.db.GetAccountsByUser(c.Request.Context(), s.userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if accounts == nil {
		accounts = make([]models.BankAccount, 0)
	}
	c.JSON(http.StatusOK, accounts)
}

func (s *Server) getTransactions(c *gin.Context) {
	txns, err := s.db.GetTransactionsByUser(c.Request.Context(), s.userID, 30, 0, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if txns == nil {
		txns = make([]models.Transaction, 0)
	}
	c.JSON(http.StatusOK, txns)
}

func (s *Server) getOverview(c *gin.Context) {
	ctx := c.Request.Context()

	accounts, _ := s.db.GetAccountsByUser(ctx, s.userID)
	totalBalance, _ := s.db.GetTotalBalance(ctx, s.userID)
	txns, _ := s.db.GetTransactionsByUser(ctx, s.userID, 30, 0, 1000)

	var totalDebit, totalCredit float64
	for _, txn := range txns {
		if txn.Type == "debit" {
			totalDebit += txn.Amount
		} else {
			totalCredit += txn.Amount
		}
	}

	c.JSON(http.StatusOK, OverviewResponse{
		TotalBalance:     totalBalance,
		TotalAccounts:    len(accounts),
		TotalDebit:       totalDebit,
		TotalCredit:      totalCredit,
		TransactionCount: len(txns),
	})
}

func (s *Server) getCategories(c *gin.Context) {
	cats, err := s.db.GetCategories(c.Request.Context(), s.userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cats)
}

func (s *Server) getBrainStatus(c *gin.Context) {
	metrics, err := s.db.GetBrainMetrics(c.Request.Context(), s.userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if metrics == nil {
		c.JSON(http.StatusOK, gin.H{"status": "no_data"})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getSpendByCategory(c *gin.Context) {
	ctx := c.Request.Context()
	txns, _ := s.db.GetTransactionsByUser(ctx, s.userID, 30, 0, 1000)

	catMap := make(map[string]*CategorySpend)
	for _, txn := range txns {
		if txn.Type != "debit" || txn.Category == "" {
			continue
		}
		if _, ok := catMap[txn.Category]; !ok {
			catMap[txn.Category] = &CategorySpend{Category: txn.Category}
		}
		catMap[txn.Category].Amount += txn.Amount
		catMap[txn.Category].Count++
	}

	result := make([]CategorySpend, 0, len(catMap))
	for _, cs := range catMap {
		result = append(result, *cs)
	}
	c.JSON(http.StatusOK, result)
}

