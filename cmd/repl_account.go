package cmd

import (
	"context"
	"fmt"
)

func (s *replState) cmdAccounts() {
	if s.db == nil {
		s.printer.Error("Not connected to database. Run 'start' first.")
		return
	}

	accounts, err := s.db.GetAccountsByUser(context.Background(), s.userID)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to fetch accounts: %v", err))
		return
	}

	if len(accounts) == 0 {
		s.printer.Info("No accounts found. Use 'account-add' to add your first account.")
		return
	}

	headers := []string{"Bank", "Account", "Type", "Balance", "Currency"}
	rows := make([][]string, 0, len(accounts))
	for _, acc := range accounts {
		rows = append(rows, []string{
			acc.BankName,
			"****" + acc.AccountNumber,
			acc.AccountType,
			formatAmount(acc.Balance),
			acc.Currency,
		})
	}
	s.printer.Table(headers, rows)
}

func (s *replState) cmdBalance() {
	if s.db == nil {
		s.printer.Error("Not connected to database. Run 'start' first.")
		return
	}

	total, err := s.db.GetTotalBalance(context.Background(), s.userID)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to get balance: %v", err))
		return
	}

	s.printer.Box("Total Balance", fmt.Sprintf("%s %s", s.cfg.CLI.CurrencySymbol, formatAmount(total)))
}

func (s *replState) cmdOverview() {
	if s.db == nil {
		s.printer.Error("Not connected to database. Run 'start' first.")
		return
	}

	accounts, err := s.db.GetAccountsByUser(context.Background(), s.userID)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to fetch accounts: %v", err))
		return
	}

	if len(accounts) == 0 {
		s.printer.Info("No accounts found. Use 'account-add' to get started.")
		return
	}

	var total float64
	lines := ""
	for _, acc := range accounts {
		total += acc.Balance
		lines += fmt.Sprintf("%s %s ****%s    %s %s\n",
			acc.BankName, acc.AccountType, acc.AccountNumber,
			s.cfg.CLI.CurrencySymbol, formatAmount(acc.Balance))
	}
	lines += fmt.Sprintf("\nTotal Balance           %s %s",
		s.cfg.CLI.CurrencySymbol, formatAmount(total))

	s.printer.Box("All Accounts Overview", lines)
}

func formatAmount(amount float64) string {
	if amount == float64(int64(amount)) {
		return formatIndian(int64(amount))
	}
	whole := int64(amount)
	frac := int64((amount - float64(whole)) * 100)
	return fmt.Sprintf("%s.%02d", formatIndian(whole), frac)
}

func formatIndian(n int64) string {
	if n < 0 {
		return "-" + formatIndian(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	result := s[len(s)-3:]
	s = s[:len(s)-3]

	for len(s) > 2 {
		result = s[len(s)-2:] + "," + result
		s = s[:len(s)-2]
	}
	if len(s) > 0 {
		result = s + "," + result
	}
	return result
}
