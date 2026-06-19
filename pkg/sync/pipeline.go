package sync

import (
	"context"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/gmail"
	"github.com/innacy/finance-agent/pkg/parsers/hdfc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type DB interface {
	CreateTransaction(ctx context.Context, txn *models.Transaction) error
	TransactionExists(ctx context.Context, userID string, accountID bson.ObjectID, date time.Time, amount float64, reference string) (bool, error)
	GetAccountsByUser(ctx context.Context, userID string) ([]models.BankAccount, error)
	CreateAccount(ctx context.Context, account *models.BankAccount) error
	UpdateBalance(ctx context.Context, id bson.ObjectID, balance float64) error
	GetSyncState(ctx context.Context, userID, source string) (*models.SyncState, error)
	UpsertSyncState(ctx context.Context, state *models.SyncState) error
}

type Result struct {
	Processed        int
	Created          int
	Duplicates       int
	Skipped          int
	Errors           int
	AccountsDetected int
}

type Pipeline struct {
	db     DB
	userID string
}

func NewPipeline(db DB, userID string) *Pipeline {
	return &Pipeline{db: db, userID: userID}
}

func (p *Pipeline) Process(ctx context.Context, emails []gmail.RawMessage) Result {
	var result Result

	for _, email := range emails {
		result.Processed++

		parsed, err := hdfc.ParseEmail(email.Subject, email.Body)
		if err != nil || parsed == nil {
			result.Skipped++
			continue
		}

		account, err := p.resolveAccount(ctx, parsed.AccountNumber)
		if err != nil {
			result.Errors++
			continue
		}
		if account == nil {
			acc, err := p.autoDetectAccount(ctx, parsed.AccountNumber)
			if err != nil {
				result.Errors++
				continue
			}
			account = acc
			result.AccountsDetected++
		}

		exists, err := p.db.TransactionExists(ctx, p.userID, account.ID, parsed.TransactionDate, parsed.Amount, parsed.Reference)
		if err != nil {
			result.Errors++
			continue
		}
		if exists {
			result.Duplicates++
			continue
		}

		txn := &models.Transaction{
			UserID:          p.userID,
			AccountID:       account.ID,
			Type:            parsed.Type,
			Amount:          parsed.Amount,
			BalanceAfter:    parsed.BalanceAfter,
			Description:     parsed.Description,
			Merchant:        parsed.Merchant,
			Channel:         parsed.Channel,
			CounterpartyUPI: parsed.CounterpartyUPI,
			Reference:       parsed.Reference,
			TransactionDate: parsed.TransactionDate,
			Source:          "gmail_alert",
			SourceRef:       email.ID,
			ReviewStatus:    "auto_accepted",
			Confidence:      1.0,
		}

		if err := p.db.CreateTransaction(ctx, txn); err != nil {
			result.Errors++
			continue
		}
		result.Created++

		if parsed.BalanceAfter > 0 {
			p.db.UpdateBalance(ctx, account.ID, parsed.BalanceAfter)
		}
	}

	return result
}

func (p *Pipeline) resolveAccount(ctx context.Context, accountNumber string) (*models.BankAccount, error) {
	accounts, err := p.db.GetAccountsByUser(ctx, p.userID)
	if err != nil {
		return nil, err
	}
	for i, acc := range accounts {
		if acc.AccountNumber == accountNumber && acc.IsActive {
			return &accounts[i], nil
		}
	}
	return nil, nil
}

func (p *Pipeline) autoDetectAccount(ctx context.Context, accountNumber string) (*models.BankAccount, error) {
	account := &models.BankAccount{
		UserID:        p.userID,
		BankName:      "HDFC",
		AccountNumber: accountNumber,
		AccountType:   "savings",
		Currency:      "INR",
		IsActive:      true,
		IsConfirmed:   false,
		DetectedFrom:  "gmail_alert",
		LastUpdated:   time.Now(),
	}

	if err := p.db.CreateAccount(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}
