package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	OpenAI struct {
		APIKey string `yaml:"api_key"`
		Model  string `yaml:"model"`
	} `yaml:"openai"`
	Telegram struct {
		BotToken string `yaml:"bot_token"`
		ChatID   int64  `yaml:"chat_id"`
	} `yaml:"telegram"`
	NewsAPI struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"newsapi"`
	Preferences struct {
		Topics      []string `yaml:"topics"`
		Languages   []string `yaml:"languages"`
		Countries   []string `yaml:"countries"`
		NumArticles int      `yaml:"num_articles"`
	} `yaml:"preferences"`
	RSS struct {
		Feeds []string `yaml:"feeds"`
	} `yaml:"rss"`
}

func Load(configFile string) (*Config, error) {
	cfg := &Config{}

	if err := loadDotEnv(configFile); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		cfg.OpenAI.APIKey = openaiKey
	}
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		cfg.OpenAI.Model = model
	}
	if cfg.OpenAI.Model == "" {
		cfg.OpenAI.Model = "gpt-4o-mini"
	}

	if telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN"); telegramToken != "" {
		cfg.Telegram.BotToken = telegramToken
	}
	if newsAPIKey := os.Getenv("NEWSAPI_API_KEY"); newsAPIKey != "" {
		cfg.NewsAPI.APIKey = newsAPIKey
	}

	if cfg.OpenAI.APIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not configured")
	}
	if cfg.Telegram.BotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN not configured")
	}

	if cfg.Preferences.NumArticles == 0 {
		cfg.Preferences.NumArticles = 5
	}
	if len(cfg.Preferences.Languages) == 0 {
		cfg.Preferences.Languages = []string{"en"}
	}

	return cfg, nil
}

func loadDotEnv(configFile string) error {
	configDir := "."
	if configFile != "" {
		configDir = filepath.Dir(configFile)
		if configDir == "" {
			configDir = "."
		}
	}

	dotEnvPath := filepath.Join(configDir, ".env")
	file, err := os.Open(dotEnvPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if value == "" {
			continue
		}

		if (strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'")) && len(value) >= 2 {
			quote := value[0]
			if end := strings.IndexRune(value[1:], rune(quote)); end >= 0 {
				value = value[1 : end+1]
			} else {
				value = value[1:]
			}
		} else {
			if idx := strings.Index(value, " #"); idx >= 0 {
				value = strings.TrimSpace(value[:idx])
			}
		}
		value = strings.Trim(value, "\"'")
		if key == "" {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}
