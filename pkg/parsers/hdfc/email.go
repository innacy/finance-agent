package hdfc

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ParsedTransaction struct {
	Type            string
	Amount          float64
	BalanceAfter    float64
	AccountNumber   string
	Merchant        string
	CounterpartyUPI string
	Reference       string
	Channel         string
	TransactionDate time.Time
	Description     string
}

var (
	reUPIDebit = regexp.MustCompile(
		`Rs\.([0-9,]+\.?\d*)\s+has been debited from account \*\*(\d{4}) on (\d{2}-\d{2}-\d{4}) to VPA ([^\s(]+)\s*\(UPI Ref No\.\s*(\d+)\)`)

	reUPICredit = regexp.MustCompile(
		`Rs\.([0-9,]+\.?\d*)\s+has been credited to your account \*\*(\d{4}) on (\d{2}-\d{2}-\d{4}) by VPA ([^\s(]+)\s*\(UPI Ref No\.\s*(\d+)\)`)

	reNEFTCredit = regexp.MustCompile(
		`Rs\.([0-9,]+\.?\d*)\s+has been credited to your A/c \*\*(\d{4}) on (\d{2}-\d{2}-\d{4}) by NEFT from (.+?)\s*\(Ref No\.\s*([^)]+)\)`)

	reATM = regexp.MustCompile(
		`Rs\.([0-9,]+\.?\d*)\s+has been withdrawn from your A/c \*\*(\d{4}) at ATM on (\d{2}-\d{2}-\d{4})\s*\(Ref No\.\s*([^)]+)\)`)

	reCreditCard = regexp.MustCompile(
		`Rs\.([0-9,]+\.?\d*)\s+has been spent on your HDFC Bank Credit Card ending (\d{4}) at (.+?) on (\d{2}-\d{2}-\d{4})`)

	reBalance = regexp.MustCompile(
		`(?:Available Balance|Available Credit Limit):\s*Rs\.([0-9,]+\.?\d*)`)

	reAccountFromSubject = regexp.MustCompile(`XX(\d{4})`)
)

func ParseEmail(subject, body string) (*ParsedTransaction, error) {
	if m := reUPIDebit.FindStringSubmatch(body); m != nil {
		return &ParsedTransaction{
			Type:            "debit",
			Amount:          parseAmount(m[1]),
			AccountNumber:   m[2],
			TransactionDate: parseDate(m[3]),
			CounterpartyUPI: m[4],
			Reference:       m[5],
			Channel:         "UPI",
			BalanceAfter:    extractBalance(body),
			Description:     buildDescription("UPI", m[4]),
		}, nil
	}

	if m := reUPICredit.FindStringSubmatch(body); m != nil {
		return &ParsedTransaction{
			Type:            "credit",
			Amount:          parseAmount(m[1]),
			AccountNumber:   m[2],
			TransactionDate: parseDate(m[3]),
			CounterpartyUPI: m[4],
			Reference:       m[5],
			Channel:         "UPI",
			BalanceAfter:    extractBalance(body),
			Description:     buildDescription("UPI", m[4]),
		}, nil
	}

	if m := reNEFTCredit.FindStringSubmatch(body); m != nil {
		return &ParsedTransaction{
			Type:            "credit",
			Amount:          parseAmount(m[1]),
			AccountNumber:   m[2],
			TransactionDate: parseDate(m[3]),
			Merchant:        strings.TrimSpace(m[4]),
			Reference:       strings.TrimSpace(m[5]),
			Channel:         "NEFT",
			BalanceAfter:    extractBalance(body),
			Description:     buildDescription("NEFT", m[4]),
		}, nil
	}

	if m := reATM.FindStringSubmatch(body); m != nil {
		return &ParsedTransaction{
			Type:            "debit",
			Amount:          parseAmount(m[1]),
			AccountNumber:   m[2],
			TransactionDate: parseDate(m[3]),
			Reference:       strings.TrimSpace(m[4]),
			Channel:         "ATM",
			BalanceAfter:    extractBalance(body),
			Description:     "ATM Withdrawal",
		}, nil
	}

	if m := reCreditCard.FindStringSubmatch(body); m != nil {
		return &ParsedTransaction{
			Type:            "debit",
			Amount:          parseAmount(m[1]),
			AccountNumber:   m[2],
			Merchant:        strings.TrimSpace(m[3]),
			TransactionDate: parseDate(m[4]),
			Channel:         "POS",
			BalanceAfter:    extractBalance(body),
			Description:     buildDescription("CARD", m[3]),
		}, nil
	}

	return nil, nil
}

func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseDate(s string) time.Time {
	t, err := time.ParseInLocation("02-01-2006", s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

func extractBalance(body string) float64 {
	if m := reBalance.FindStringSubmatch(body); m != nil {
		return parseAmount(m[1])
	}
	return 0
}

func buildDescription(channel, detail string) string {
	return channel + "-" + strings.TrimSpace(detail)
}
