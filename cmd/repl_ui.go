package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/innacy/finance-agent/pkg/api"
)

func (s *replState) cmdUIStart() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	if s.uiServer != nil {
		s.printer.Warn("UI server is already running.")
		return
	}

	srv := api.NewServer(s.db, s.userID)
	srv.ServeStatic("web/dist")

	port := 8090
	s.uiServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: srv.Router(),
	}

	go func() {
		if err := s.uiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.printer.Error(fmt.Sprintf("UI server error: %v", err))
		}
	}()

	s.printer.Success(fmt.Sprintf("UI started at http://localhost:%d", port))
}

func (s *replState) cmdUIStop() {
	if s.uiServer == nil {
		s.printer.Warn("UI server is not running.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.uiServer.Shutdown(ctx)
	s.uiServer = nil
	s.printer.Success("UI server stopped.")
}
