package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/config"
	"github.com/innacy/finance-agent/pkg/db"
	"github.com/innacy/finance-agent/pkg/output"
)

func newTestStateWithDB(t *testing.T) (*replState, *bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	cfg, _ := config.Load("")
	printer := output.NewPrinter(&buf, false)

	dbCfg := &config.DBConfig{
		URI:      "mongodb://localhost:27017",
		Database: "finance-agent-cmd-test",
		Timeout:  5 * time.Second,
	}
	client, err := db.NewClient(dbCfg)
	if err != nil {
		t.Fatalf("failed to create db client: %v", err)
	}
	if err := client.Ping(context.Background()); err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	state := &replState{
		cfg:     cfg,
		printer: printer,
		output:  &buf,
		db:      client,
		userID:  "test-user",
	}

	cleanup := func() {
		client.DB().Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return state, &buf, cleanup
}

func TestCmdAccountsEmpty(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdAccounts()

	out := buf.String()
	if !strings.Contains(out, "No accounts") || !strings.Contains(out, "account-add") {
		t.Errorf("empty accounts should show hint to add, got: %q", out)
	}
}

func TestCmdAccountsShowsList(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.CreateAccount(ctx, &models.BankAccount{
		UserID: "test-user", BankName: "HDFC", AccountNumber: "4521",
		AccountType: "savings", Balance: 124350, Currency: "INR",
		IsActive: true, IsConfirmed: true,
	})
	state.db.CreateAccount(ctx, &models.BankAccount{
		UserID: "test-user", BankName: "HDFC", AccountNumber: "8903",
		AccountType: "salary", Balance: 45200, Currency: "INR",
		IsActive: true, IsConfirmed: true,
	})

	state.cmdAccounts()

	out := buf.String()
	if !strings.Contains(out, "4521") {
		t.Error("accounts should show account number 4521")
	}
	if !strings.Contains(out, "8903") {
		t.Error("accounts should show account number 8903")
	}
	if !strings.Contains(out, "HDFC") {
		t.Error("accounts should show bank name")
	}
}

func TestCmdBalance(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.CreateAccount(ctx, &models.BankAccount{
		UserID: "test-user", BankName: "HDFC", AccountNumber: "1111",
		Balance: 100000, Currency: "INR", IsActive: true, IsConfirmed: true,
	})
	state.db.CreateAccount(ctx, &models.BankAccount{
		UserID: "test-user", BankName: "HDFC", AccountNumber: "2222",
		Balance: 50000, Currency: "INR", IsActive: true, IsConfirmed: true,
	})

	state.cmdBalance()

	out := buf.String()
	if !strings.Contains(out, "1,50,000") {
		t.Errorf("balance should show total in Indian format (1,50,000), got: %q", out)
	}
}

func TestCmdOverview(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.CreateAccount(ctx, &models.BankAccount{
		UserID: "test-user", BankName: "HDFC", AccountNumber: "4521",
		AccountType: "savings", Balance: 124350, Currency: "INR",
		IsActive: true, IsConfirmed: true,
	})

	state.cmdOverview()

	out := buf.String()
	if !strings.Contains(out, "HDFC") {
		t.Error("overview should contain bank name")
	}
	if !strings.Contains(out, "4521") {
		t.Error("overview should contain account number")
	}
}

func TestDispatchAccountsCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("accounts")
	if result != Continue {
		t.Error("accounts command should return Continue")
	}
}

func TestDispatchBalanceCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("balance")
	if result != Continue {
		t.Error("balance command should return Continue")
	}
}

func TestDispatchOverviewCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("overview")
	if result != Continue {
		t.Error("overview command should return Continue")
	}
}
