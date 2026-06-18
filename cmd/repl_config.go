package cmd

import "fmt"

func (s *replState) cmdConfig() {
	fmt.Fprintf(s.output, "╭──────────────────────────────────────╮\n")
	fmt.Fprintf(s.output, "│         Configuration                │\n")
	fmt.Fprintf(s.output, "├──────────────────────────────────────┤\n")
	fmt.Fprintf(s.output, "│  DB URI:      %s\n", s.cfg.DB.URI)
	fmt.Fprintf(s.output, "│  DB Name:     %s\n", s.cfg.DB.Database)
	fmt.Fprintf(s.output, "│  AI Provider: %s\n", s.cfg.AI.Provider)
	fmt.Fprintf(s.output, "│  AI Model:    %s\n", s.cfg.AI.Model)
	fmt.Fprintf(s.output, "│  Theme:       %s\n", s.cfg.CLI.Theme)
	fmt.Fprintf(s.output, "│  Currency:    %s\n", s.cfg.CLI.CurrencySymbol)
	fmt.Fprintf(s.output, "│  Bank:        %s\n", s.cfg.Parsers.DefaultBank)
	fmt.Fprintf(s.output, "╰──────────────────────────────────────╯\n")
}
