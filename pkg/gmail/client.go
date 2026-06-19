package gmail

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrAuthExpired     = errors.New("gmail auth expired or invalid")
)

var bankSenders = []string{
	"alerts@hdfcbank.net",
	"alerts@icicibank.com",
	"alerts@axisbank.com",
}

type RawMessage struct {
	ID      string
	Subject string
	Body    string
	From    string
	Date    time.Time
}

type GmailAPI interface {
	ListMessages(ctx context.Context, query string, maxResults int64) ([]string, error)
	GetMessage(ctx context.Context, id string) (*RawMessage, error)
}

type Client struct {
	api GmailAPI
}

func NewClient(api GmailAPI) *Client {
	return &Client{api: api}
}

func (c *Client) FetchFinanceEmails(ctx context.Context, since time.Time, maxResults int64) ([]RawMessage, error) {
	query := BuildQuery(since)

	ids, err := c.api.ListMessages(ctx, query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}

	messages := make([]RawMessage, 0, len(ids))
	for _, id := range ids {
		msg, err := c.api.GetMessage(ctx, id)
		if err != nil {
			continue
		}
		messages = append(messages, *msg)
	}

	return messages, nil
}

func BuildQuery(since time.Time) string {
	parts := []string{
		fmt.Sprintf("from:(%s)", strings.Join(bankSenders, " OR ")),
		"subject:(alert OR transaction OR debited OR credited)",
	}

	if !since.IsZero() {
		parts = append(parts, fmt.Sprintf("after:%s", since.Format("2006/01/02")))
	}

	return strings.Join(parts, " ")
}
