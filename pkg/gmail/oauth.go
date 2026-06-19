package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type OAuthConfig struct {
	CredentialsFile string
	TokenFile       string
}

func Authenticate(ctx context.Context, cfg OAuthConfig) (*gmail.Service, error) {
	credBytes, err := os.ReadFile(cfg.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	oauthCfg, err := google.ConfigFromJSON(credBytes, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	token, err := loadToken(cfg.TokenFile)
	if err != nil {
		token, err = getTokenFromWeb(ctx, oauthCfg)
		if err != nil {
			return nil, fmt.Errorf("obtaining token: %w", err)
		}
		if err := saveToken(cfg.TokenFile, token); err != nil {
			return nil, fmt.Errorf("saving token: %w", err)
		}
	}

	client := oauthCfg.Client(ctx, token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("creating gmail service: %w", err)
	}

	return service, nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = "http://localhost:8089/callback"

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\n  Open this URL in your browser:\n  %s\n\n", authURL)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	server := &http.Server{Addr: ":8089"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			return
		}
		fmt.Fprintf(w, "<h1>Authentication successful!</h1><p>You can close this tab.</p>")
		codeCh <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("authentication timed out")
	}

	server.Shutdown(ctx)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}
	return token, nil
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func saveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
