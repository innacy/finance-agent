package hdfc

import (
	"testing"
	"time"
)

func TestParseUPIDebit(t *testing.T) {
	body := `Dear Customer,
Rs.450.00 has been debited from account **4521 on 15-06-2026 to VPA swiggy@axisbank (UPI Ref No. 415678901234).
Available Balance: Rs.99,550.00`

	result, err := ParseEmail("Alert : Update for your HDFC Bank A/c XX4521", body)
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Type != "debit" {
		t.Errorf("expected debit, got %q", result.Type)
	}
	if result.Amount != 450.00 {
		t.Errorf("expected 450.00, got %f", result.Amount)
	}
	if result.AccountNumber != "4521" {
		t.Errorf("expected account 4521, got %q", result.AccountNumber)
	}
	if result.CounterpartyUPI != "swiggy@axisbank" {
		t.Errorf("expected swiggy@axisbank, got %q", result.CounterpartyUPI)
	}
	if result.Reference != "415678901234" {
		t.Errorf("expected ref 415678901234, got %q", result.Reference)
	}
	if result.BalanceAfter != 99550.00 {
		t.Errorf("expected balance 99550, got %f", result.BalanceAfter)
	}
	if result.Channel != "UPI" {
		t.Errorf("expected UPI channel, got %q", result.Channel)
	}
}

func TestParseUPICredit(t *testing.T) {
	body := `Dear Customer,
Rs.15,000.00 has been credited to your account **8903 on 15-06-2026 by VPA ravi.kumar@okicici (UPI Ref No. 415678905678).
Available Balance: Rs.60,200.00`

	result, err := ParseEmail("Alert : Update for your HDFC Bank A/c XX8903", body)
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}
	if result.Type != "credit" {
		t.Errorf("expected credit, got %q", result.Type)
	}
	if result.Amount != 15000.00 {
		t.Errorf("expected 15000, got %f", result.Amount)
	}
	if result.AccountNumber != "8903" {
		t.Errorf("expected account 8903, got %q", result.AccountNumber)
	}
	if result.CounterpartyUPI != "ravi.kumar@okicici" {
		t.Errorf("expected ravi.kumar@okicici, got %q", result.CounterpartyUPI)
	}
	if result.BalanceAfter != 60200.00 {
		t.Errorf("expected balance 60200, got %f", result.BalanceAfter)
	}
}

func TestParseNEFTCredit(t *testing.T) {
	body := `Dear Customer,
Rs.85,000.00 has been credited to your A/c **8903 on 01-06-2026 by NEFT from ACME CORP LTD (Ref No. NEFT-N123456789).
Available Balance: Rs.1,45,200.00`

	result, err := ParseEmail("Alert : Rs.85,000.00 credited to your HDFC Bank A/c XX8903", body)
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}
	if result.Type != "credit" {
		t.Errorf("expected credit, got %q", result.Type)
	}
	if result.Amount != 85000.00 {
		t.Errorf("expected 85000, got %f", result.Amount)
	}
	if result.Channel != "NEFT" {
		t.Errorf("expected NEFT channel, got %q", result.Channel)
	}
	if result.Reference != "NEFT-N123456789" {
		t.Errorf("expected ref NEFT-N123456789, got %q", result.Reference)
	}
	if result.BalanceAfter != 145200.00 {
		t.Errorf("expected balance 145200, got %f", result.BalanceAfter)
	}
}

func TestParseATMWithdrawal(t *testing.T) {
	body := `Dear Customer,
Rs.10,000.00 has been withdrawn from your A/c **4521 at ATM on 10-06-2026 (Ref No. ATM-98765).
Available Balance: Rs.89,550.00`

	result, err := ParseEmail("Alert : Rs.10,000.00 withdrawn from ATM for your HDFC Bank A/c", body)
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}
	if result.Type != "debit" {
		t.Errorf("expected debit, got %q", result.Type)
	}
	if result.Amount != 10000.00 {
		t.Errorf("expected 10000, got %f", result.Amount)
	}
	if result.Channel != "ATM" {
		t.Errorf("expected ATM channel, got %q", result.Channel)
	}
}

func TestParseCreditCardTransaction(t *testing.T) {
	body := `Dear Customer,
Rs.3,450.00 has been spent on your HDFC Bank Credit Card ending 7788 at DECATHLON SPORTS on 12-06-2026.
Available Credit Limit: Rs.2,96,550.00`

	result, err := ParseEmail("Alert : HDFC Bank Credit Card XX7788 transaction", body)
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}
	if result.Type != "debit" {
		t.Errorf("expected debit, got %q", result.Type)
	}
	if result.Amount != 3450.00 {
		t.Errorf("expected 3450, got %f", result.Amount)
	}
	if result.Merchant != "DECATHLON SPORTS" {
		t.Errorf("expected DECATHLON SPORTS, got %q", result.Merchant)
	}
	if result.AccountNumber != "7788" {
		t.Errorf("expected card 7788, got %q", result.AccountNumber)
	}
	if result.Channel != "POS" {
		t.Errorf("expected POS channel, got %q", result.Channel)
	}
}

func TestParseTransactionDate(t *testing.T) {
	body := `Dear Customer,
Rs.100.00 has been debited from account **4521 on 25-12-2026 to VPA test@upi (UPI Ref No. 123456789012).
Available Balance: Rs.50,000.00`

	result, _ := ParseEmail("Alert : Update for your HDFC Bank A/c XX4521", body)
	if result == nil {
		t.Fatal("expected result")
	}

	expected := time.Date(2026, 12, 25, 0, 0, 0, 0, time.Local)
	if !result.TransactionDate.Equal(expected) {
		t.Errorf("expected date %v, got %v", expected, result.TransactionDate)
	}
}

func TestParseUnknownEmailReturnsNil(t *testing.T) {
	result, err := ParseEmail("Weekly Newsletter", "Hello, here's your weekly update...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("unknown email format should return nil")
	}
}

func TestParseAmountWithCommas(t *testing.T) {
	body := `Dear Customer,
Rs.1,23,456.78 has been debited from account **4521 on 15-06-2026 to VPA merchant@upi (UPI Ref No. 999888777666).
Available Balance: Rs.8,76,543.22`

	result, _ := ParseEmail("Alert : Update for your HDFC Bank A/c XX4521", body)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Amount != 123456.78 {
		t.Errorf("expected 123456.78, got %f", result.Amount)
	}
	if result.BalanceAfter != 876543.22 {
		t.Errorf("expected 876543.22, got %f", result.BalanceAfter)
	}
}
