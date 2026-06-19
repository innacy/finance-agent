package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

type GoogleGmailAPI struct {
	service *gmail.Service
}

func NewGoogleGmailAPI(service *gmail.Service) *GoogleGmailAPI {
	return &GoogleGmailAPI{service: service}
}

func (g *GoogleGmailAPI) ListMessages(ctx context.Context, query string, maxResults int64) ([]string, error) {
	call := g.service.Users.Messages.List("me").Q(query).MaxResults(maxResults)

	var ids []string
	err := call.Pages(ctx, func(resp *gmail.ListMessagesResponse) error {
		for _, msg := range resp.Messages {
			ids = append(ids, msg.Id)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing gmail messages: %w", err)
	}
	return ids, nil
}

func (g *GoogleGmailAPI) GetMessage(ctx context.Context, id string) (*RawMessage, error) {
	msg, err := g.service.Users.Messages.Get("me", id).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("getting message %s: %w", id, err)
	}

	raw := &RawMessage{ID: id}

	for _, header := range msg.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "subject":
			raw.Subject = header.Value
		case "from":
			raw.From = header.Value
		case "date":
			raw.Date = parseEmailDate(header.Value)
		}
	}

	raw.Body = extractBody(msg.Payload)
	return raw, nil
}

func extractBody(payload *gmail.MessagePart) string {
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err == nil {
				return string(data)
			}
		}
	}

	for _, part := range payload.Parts {
		if body := extractBody(part); body != "" {
			return body
		}
	}

	return ""
}

func parseEmailDate(s string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
