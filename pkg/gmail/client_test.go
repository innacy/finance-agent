package gmail

import (
	"context"
	"testing"
	"time"
)

type mockGmailAPI struct {
	messages []RawMessage
	fetchErr error
}

func (m *mockGmailAPI) ListMessages(ctx context.Context, query string, maxResults int64) ([]string, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	ids := make([]string, len(m.messages))
	for i, msg := range m.messages {
		ids[i] = msg.ID
	}
	return ids, nil
}

func (m *mockGmailAPI) GetMessage(ctx context.Context, id string) (*RawMessage, error) {
	for _, msg := range m.messages {
		if msg.ID == id {
			return &msg, nil
		}
	}
	return nil, ErrMessageNotFound
}

func TestFetchFinanceEmails(t *testing.T) {
	api := &mockGmailAPI{
		messages: []RawMessage{
			{
				ID:      "msg-1",
				Subject: "Alert : Update for your HDFC Bank A/c XX4521",
				Body:    "Rs.450.00 has been debited...",
				From:    "alerts@hdfcbank.net",
				Date:    time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:      "msg-2",
				Subject: "Your Amazon order",
				Body:    "Your order has been shipped...",
				From:    "shipments@amazon.in",
				Date:    time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
			},
		},
	}

	client := NewClient(api)
	emails, err := client.FetchFinanceEmails(context.Background(), time.Time{}, 100)
	if err != nil {
		t.Fatalf("FetchFinanceEmails failed: %v", err)
	}
	if len(emails) != 2 {
		t.Errorf("expected 2 emails, got %d", len(emails))
	}
}

func TestFetchFinanceEmailsSince(t *testing.T) {
	api := &mockGmailAPI{
		messages: []RawMessage{
			{
				ID:      "msg-1",
				Subject: "HDFC Alert",
				From:    "alerts@hdfcbank.net",
				Date:    time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	client := NewClient(api)
	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	emails, err := client.FetchFinanceEmails(context.Background(), since, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(emails) != 1 {
		t.Errorf("expected 1 email, got %d", len(emails))
	}
}

func TestFetchReturnsErrorOnAPIFailure(t *testing.T) {
	api := &mockGmailAPI{
		fetchErr: ErrAuthExpired,
	}

	client := NewClient(api)
	_, err := client.FetchFinanceEmails(context.Background(), time.Time{}, 100)
	if err == nil {
		t.Error("expected error on API failure")
	}
}

func TestBuildQueryString(t *testing.T) {
	tests := []struct {
		name     string
		since    time.Time
		wantSub  string
		wantFull string
	}{
		{
			name:    "no since",
			since:   time.Time{},
			wantSub: "from:(alerts@hdfcbank.net OR alerts@icicibank.com OR alerts@axisbank.com)",
		},
		{
			name:    "with since",
			since:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			wantSub: "after:2026/06/01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := BuildQuery(tt.since)
			if tt.wantSub != "" && !contains(q, tt.wantSub) {
				t.Errorf("query %q missing expected substring %q", q, tt.wantSub)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
