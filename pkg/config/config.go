package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DB         DBConfig         `mapstructure:"db"`
	Gmail      GmailConfig      `mapstructure:"gmail"`
	Daemon     DaemonConfig     `mapstructure:"daemon"`
	AI         AIConfig         `mapstructure:"ai"`
	CLI        CLIConfig        `mapstructure:"cli"`
	Parsers    ParsersConfig    `mapstructure:"parsers"`
	Categories CategoriesConfig `mapstructure:"categories"`
}

type DBConfig struct {
	URI      string        `mapstructure:"uri"`
	Database string        `mapstructure:"database"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

type GmailConfig struct {
	CredentialsFile string   `mapstructure:"credentials_file"`
	TokenFile       string   `mapstructure:"token_file"`
	User            string   `mapstructure:"user"`
	Labels          []string `mapstructure:"labels"`
	SenderFilters   []string `mapstructure:"sender_filters"`
}

type DaemonConfig struct {
	PollInterval time.Duration `mapstructure:"poll_interval"`
	BatchSize    int           `mapstructure:"batch_size"`
	HealthPort   int           `mapstructure:"health_port"`
}

type AIConfig struct {
	Provider  string `mapstructure:"provider"`
	APIKeyEnv string `mapstructure:"api_key_env"`
	Model     string `mapstructure:"model"`
	MaxBatch  int    `mapstructure:"max_batch"`
}

type CLIConfig struct {
	Theme          string `mapstructure:"theme"`
	CurrencySymbol string `mapstructure:"currency_symbol"`
	DateFormat     string `mapstructure:"date_format"`
	ConfirmPrompts bool   `mapstructure:"confirm_prompts"`
	Verbose        bool   `mapstructure:"verbose"`
}

type ParsersConfig struct {
	DefaultBank       string `mapstructure:"default_bank"`
	StatementUploadDir string `mapstructure:"statement_upload_dir"`
}

type CategoriesConfig struct {
	AutoLearn     bool    `mapstructure:"auto_learn"`
	MinConfidence float64 `mapstructure:"min_confidence"`
	AIThreshold   float64 `mapstructure:"ai_threshold"`
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("db.uri", "mongodb://localhost:27017")
	v.SetDefault("db.database", "finance-agent")
	v.SetDefault("db.timeout", "10s")

	v.SetDefault("gmail.credentials_file", "./credentials.json")
	v.SetDefault("gmail.token_file", "./token.json")
	v.SetDefault("gmail.user", "me")
	v.SetDefault("gmail.labels", []string{"INBOX"})
	v.SetDefault("gmail.sender_filters", []string{"alerts@hdfcbank.net", "creditcards@hdfcbank.net"})

	v.SetDefault("daemon.poll_interval", "5m")
	v.SetDefault("daemon.batch_size", 50)
	v.SetDefault("daemon.health_port", 9090)

	v.SetDefault("ai.provider", "nvidia")
	v.SetDefault("ai.api_key_env", "NVIDIA_API_KEY")
	v.SetDefault("ai.model", "meta/llama-3.1-70b-instruct")
	v.SetDefault("ai.max_batch", 10)

	v.SetDefault("cli.theme", "default")
	v.SetDefault("cli.currency_symbol", "₹")
	v.SetDefault("cli.date_format", "02 Jan 2006")
	v.SetDefault("cli.confirm_prompts", true)
	v.SetDefault("cli.verbose", false)

	v.SetDefault("parsers.default_bank", "hdfc")
	v.SetDefault("parsers.statement_upload_dir", "./statements")

	v.SetDefault("categories.auto_learn", true)
	v.SetDefault("categories.min_confidence", 0.8)
	v.SetDefault("categories.ai_threshold", 0.6)
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix("FINANCE_AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		_ = v.ReadInConfig()
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func Validate(cfg *Config) error {
	if cfg.DB.URI == "" {
		return fmt.Errorf("db.uri is required")
	}
	if cfg.DB.Database == "" {
		return fmt.Errorf("db.database is required")
	}
	return nil
}
