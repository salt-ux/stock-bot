package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	App       AppConfig
	DB        DBConfig
	Redis     RedisConfig
	Broker    BrokerConfig
	Market    MarketConfig
	Trading   TradingConfig
	Scheduler SchedulerConfig
	Kiwoom    KiwoomConfig
}

type AppConfig struct {
	Env  string
	Port int
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		d.User,
		url.QueryEscape(d.Password),
		d.Host,
		d.Port,
		d.Name,
	)
}

type RedisConfig struct {
	Host string
	Port int
}

type BrokerConfig struct {
	Provider string
}

type MarketConfig struct {
	Provider          string
	CacheTTLSeconds   int
	QuotePath         string
	QuoteAPIID        string
	CandlePath        string
	CandleMinuteAPIID string
	CandleDailyAPIID  string
	SymbolPath        string
	SymbolListAPIID   string
	SymbolSearchAPIID string
	SymbolCacheTTL    int
	UseWebSocket      bool
	WebSocketURL      string
	WebSocketPath     string
	WebSocketQuoteTR  string
}

type TradingConfig struct {
	InitialCash            float64
	MaxPositionNotional    float64
	DailyLossLimit         float64
	DuplicateWindowSeconds int
}

type SchedulerConfig struct {
	Enabled            bool
	Timezone           string
	MarketOpenCron     string
	AutoTradeCron      string
	AutoTradeSymbol    string
	AutoTradeInterval  string
	AutoTradeLimit     int
	AutoTradeStrategy  string
	AutoTradeQty       int64
	SMAShortWindow     int
	SMALongWindow      int
	InfiniteBuyCount   int
	InfiniteAvgPrice   float64
	InfiniteAllocation float64
}

type KiwoomConfig struct {
	BaseURL   string
	TokenPath string
	AppKey    string
	AppSecret string
}

func (a AppConfig) Address() string {
	return fmt.Sprintf(":%d", a.Port)
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Env:  getEnvOrDefault("APP_ENV", "local"),
			Port: getIntOrDefault("APP_PORT", 8080),
		},
		DB: DBConfig{
			Host:     getEnvOrDefault("DB_HOST", "127.0.0.1"),
			Port:     getIntOrDefault("DB_PORT", 3306),
			User:     getEnvOrDefault("DB_USER", "stock"),
			Password: getEnvOrDefault("DB_PASSWORD", "stockpass"),
			Name:     getEnvOrDefault("DB_NAME", "stockbot"),
		},
		Redis: RedisConfig{
			Host: getEnvOrDefault("REDIS_HOST", "127.0.0.1"),
			Port: getIntOrDefault("REDIS_PORT", 6379),
		},
		Broker: BrokerConfig{
			Provider: getEnvOrDefault("BROKER_PROVIDER", "kiwoom"),
		},
		Market: MarketConfig{
			Provider:          getEnvOrDefault("MARKET_PROVIDER", "mock"),
			CacheTTLSeconds:   getIntOrDefault("MARKET_CACHE_TTL_SECONDS", 5),
			QuotePath:         getEnvOrDefault("KIWOOM_MARKET_QUOTE_PATH", "/api/dostk/stkinfo"),
			QuoteAPIID:        getEnvOrDefault("KIWOOM_MARKET_QUOTE_API_ID", "ka10001"),
			CandlePath:        getEnvOrDefault("KIWOOM_MARKET_CANDLE_PATH", "/api/dostk/chart"),
			CandleMinuteAPIID: getEnvOrDefault("KIWOOM_MARKET_CANDLE_MINUTE_API_ID", getEnvOrDefault("KIWOOM_MARKET_CANDLE_API_ID", "ka10080")),
			CandleDailyAPIID:  getEnvOrDefault("KIWOOM_MARKET_CANDLE_DAILY_API_ID", "ka10081"),
			SymbolPath:        getEnvOrDefault("KIWOOM_MARKET_SYMBOL_PATH", getEnvOrDefault("KIWOOM_MARKET_QUOTE_PATH", "/api/dostk/stkinfo")),
			SymbolListAPIID:   getEnvOrDefault("KIWOOM_MARKET_SYMBOL_LIST_API_ID", "ka10099"),
			SymbolSearchAPIID: getEnvOrDefault("KIWOOM_MARKET_SYMBOL_SEARCH_API_ID", "ka10100"),
			SymbolCacheTTL:    getIntOrDefault("KIWOOM_MARKET_SYMBOL_CACHE_TTL_SECONDS", 600),
			UseWebSocket:      getBoolOrDefault("KIWOOM_MARKET_USE_WEBSOCKET", true),
			WebSocketURL:      getEnvOrDefault("KIWOOM_MARKET_WS_URL", ""),
			WebSocketPath:     getEnvOrDefault("KIWOOM_MARKET_WS_PATH", "/api/dostk/websocket"),
			WebSocketQuoteTR:  strings.ToUpper(getEnvOrDefault("KIWOOM_MARKET_WS_QUOTE_TR", "0B")),
		},
		Trading: TradingConfig{
			InitialCash:            getFloatOrDefault("TRADING_INITIAL_CASH", 50000000),
			MaxPositionNotional:    getFloatOrDefault("RISK_MAX_POSITION_NOTIONAL", 10000000),
			DailyLossLimit:         getFloatOrDefault("RISK_DAILY_LOSS_LIMIT", 1000000),
			DuplicateWindowSeconds: getIntOrDefault("RISK_DUPLICATE_WINDOW_SECONDS", 30),
		},
		Scheduler: SchedulerConfig{
			Enabled:            getBoolOrDefault("SCHEDULER_ENABLED", false),
			Timezone:           getEnvOrDefault("SCHEDULER_TIMEZONE", "Asia/Seoul"),
			MarketOpenCron:     getEnvOrDefault("SCHEDULER_MARKET_OPEN_CRON", "0 9 * * 1-5"),
			AutoTradeCron:      getEnvOrDefault("SCHEDULER_AUTOTRADE_CRON", "30 10 * * 1-5"),
			AutoTradeSymbol:    getEnvOrDefault("SCHEDULER_AUTOTRADE_SYMBOL", "005930"),
			AutoTradeInterval:  getEnvOrDefault("SCHEDULER_AUTOTRADE_INTERVAL", "1d"),
			AutoTradeLimit:     getIntOrDefault("SCHEDULER_AUTOTRADE_LIMIT", 60),
			AutoTradeStrategy:  strings.ToLower(getEnvOrDefault("SCHEDULER_AUTOTRADE_STRATEGY", "infinite_buy")),
			AutoTradeQty:       int64(getIntOrDefault("SCHEDULER_AUTOTRADE_QTY", 1)),
			SMAShortWindow:     getIntOrDefault("SCHEDULER_SMA_SHORT", 5),
			SMALongWindow:      getIntOrDefault("SCHEDULER_SMA_LONG", 20),
			InfiniteBuyCount:   getIntOrDefault("SCHEDULER_INFINITE_BUY_COUNT", 0),
			InfiniteAvgPrice:   getFloatOrDefault("SCHEDULER_INFINITE_AVG_PRICE", 0),
			InfiniteAllocation: getFloatOrDefault("SCHEDULER_INFINITE_ALLOCATION", 0),
		},
		Kiwoom: KiwoomConfig{
			BaseURL:   getEnvOrDefault("KIWOOM_BASE_URL", ""),
			TokenPath: getEnvOrDefault("KIWOOM_TOKEN_PATH", "/oauth2/token"),
			AppKey:    getEnvOrDefault("KIWOOM_APP_KEY", getEnvOrDefault("APP_KEY", "")),
			AppSecret: getEnvOrDefault("KIWOOM_APP_SECRET", getEnvOrDefault("APP_SEC", "")),
		},
	}

	if err := validatePort("APP_PORT", cfg.App.Port); err != nil {
		return Config{}, err
	}
	if err := validatePort("DB_PORT", cfg.DB.Port); err != nil {
		return Config{}, err
	}
	if err := validatePort("REDIS_PORT", cfg.Redis.Port); err != nil {
		return Config{}, err
	}
	if err := validateBroker(cfg.Broker); err != nil {
		return Config{}, err
	}
	if err := validateMarket(cfg.Market); err != nil {
		return Config{}, err
	}
	if err := validateTrading(cfg.Trading); err != nil {
		return Config{}, err
	}
	if err := validateScheduler(cfg.Scheduler); err != nil {
		return Config{}, err
	}
	if err := validateKiwoom(cfg.Kiwoom); err != nil {
		return Config{}, err
	}
	if strings.EqualFold(cfg.Market.Provider, "kiwoom") {
		if strings.TrimSpace(cfg.Kiwoom.BaseURL) == "" {
			return Config{}, fmt.Errorf("KIWOOM_BASE_URL is required when MARKET_PROVIDER=kiwoom")
		}
		if strings.TrimSpace(cfg.Kiwoom.AppKey) == "" {
			return Config{}, fmt.Errorf("KIWOOM_APP_KEY is required when MARKET_PROVIDER=kiwoom")
		}
		if strings.TrimSpace(cfg.Kiwoom.AppSecret) == "" {
			return Config{}, fmt.Errorf("KIWOOM_APP_SECRET is required when MARKET_PROVIDER=kiwoom")
		}
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBoolOrDefault(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func getFloatOrDefault(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func validatePort(key string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", key)
	}
	return nil
}

func validateKiwoom(cfg KiwoomConfig) error {
	if cfg.BaseURL == "" && cfg.AppKey == "" && cfg.AppSecret == "" {
		return nil
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("KIWOOM_BASE_URL is required when using kiwoom credentials")
	}
	if cfg.AppKey == "" {
		return fmt.Errorf("KIWOOM_APP_KEY is required")
	}
	if cfg.AppSecret == "" {
		return fmt.Errorf("KIWOOM_APP_SECRET is required")
	}
	if len(strings.TrimSpace(cfg.AppKey)) < 8 {
		return fmt.Errorf("KIWOOM_APP_KEY is too short")
	}
	if len(strings.TrimSpace(cfg.AppSecret)) < 8 {
		return fmt.Errorf("KIWOOM_APP_SECRET is too short")
	}
	return nil
}

func validateBroker(cfg BrokerConfig) error {
	switch cfg.Provider {
	case "kiwoom":
		return nil
	default:
		return fmt.Errorf("unsupported BROKER_PROVIDER: %s", cfg.Provider)
	}
}

func validateMarket(cfg MarketConfig) error {
	switch cfg.Provider {
	case "mock", "kiwoom":
	default:
		return fmt.Errorf("unsupported MARKET_PROVIDER: %s", cfg.Provider)
	}
	if cfg.CacheTTLSeconds < 1 || cfg.CacheTTLSeconds > 3600 {
		return fmt.Errorf("MARKET_CACHE_TTL_SECONDS must be between 1 and 3600")
	}
	if cfg.Provider == "kiwoom" {
		if strings.TrimSpace(cfg.QuotePath) == "" {
			return fmt.Errorf("KIWOOM_MARKET_QUOTE_PATH is required")
		}
		if strings.TrimSpace(cfg.QuoteAPIID) == "" {
			return fmt.Errorf("KIWOOM_MARKET_QUOTE_API_ID is required")
		}
		if strings.TrimSpace(cfg.CandlePath) == "" {
			return fmt.Errorf("KIWOOM_MARKET_CANDLE_PATH is required")
		}
		if strings.TrimSpace(cfg.CandleMinuteAPIID) == "" {
			return fmt.Errorf("KIWOOM_MARKET_CANDLE_MINUTE_API_ID is required")
		}
		if strings.TrimSpace(cfg.CandleDailyAPIID) == "" {
			return fmt.Errorf("KIWOOM_MARKET_CANDLE_DAILY_API_ID is required")
		}
		if strings.TrimSpace(cfg.SymbolPath) == "" {
			return fmt.Errorf("KIWOOM_MARKET_SYMBOL_PATH is required")
		}
		if strings.TrimSpace(cfg.SymbolListAPIID) == "" && strings.TrimSpace(cfg.SymbolSearchAPIID) == "" {
			return fmt.Errorf("KIWOOM_MARKET_SYMBOL_LIST_API_ID or KIWOOM_MARKET_SYMBOL_SEARCH_API_ID is required")
		}
		if cfg.SymbolCacheTTL < 10 || cfg.SymbolCacheTTL > 86400 {
			return fmt.Errorf("KIWOOM_MARKET_SYMBOL_CACHE_TTL_SECONDS must be between 10 and 86400")
		}
		if cfg.UseWebSocket && strings.TrimSpace(cfg.WebSocketPath) == "" && strings.TrimSpace(cfg.WebSocketURL) == "" {
			return fmt.Errorf("KIWOOM_MARKET_WS_PATH or KIWOOM_MARKET_WS_URL is required when websocket is enabled")
		}
		if cfg.UseWebSocket && strings.TrimSpace(cfg.WebSocketQuoteTR) == "" {
			return fmt.Errorf("KIWOOM_MARKET_WS_QUOTE_TR is required when websocket is enabled")
		}
	}
	return nil
}

func validateTrading(cfg TradingConfig) error {
	if cfg.InitialCash <= 0 {
		return fmt.Errorf("TRADING_INITIAL_CASH must be > 0")
	}
	if cfg.MaxPositionNotional <= 0 {
		return fmt.Errorf("RISK_MAX_POSITION_NOTIONAL must be > 0")
	}
	if cfg.DailyLossLimit <= 0 {
		return fmt.Errorf("RISK_DAILY_LOSS_LIMIT must be > 0")
	}
	if cfg.DuplicateWindowSeconds < 1 || cfg.DuplicateWindowSeconds > 3600 {
		return fmt.Errorf("RISK_DUPLICATE_WINDOW_SECONDS must be between 1 and 3600")
	}
	return nil
}

func validateScheduler(cfg SchedulerConfig) error {
	if strings.TrimSpace(cfg.Timezone) == "" {
		return fmt.Errorf("SCHEDULER_TIMEZONE is required")
	}
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.MarketOpenCron) == "" {
		return fmt.Errorf("SCHEDULER_MARKET_OPEN_CRON is required when SCHEDULER_ENABLED=true")
	}
	if strings.TrimSpace(cfg.AutoTradeCron) == "" {
		return fmt.Errorf("SCHEDULER_AUTOTRADE_CRON is required when SCHEDULER_ENABLED=true")
	}
	if strings.TrimSpace(cfg.AutoTradeSymbol) == "" {
		return fmt.Errorf("SCHEDULER_AUTOTRADE_SYMBOL is required when SCHEDULER_ENABLED=true")
	}
	if strings.TrimSpace(cfg.AutoTradeInterval) == "" {
		return fmt.Errorf("SCHEDULER_AUTOTRADE_INTERVAL is required when SCHEDULER_ENABLED=true")
	}
	if cfg.AutoTradeLimit < 2 {
		return fmt.Errorf("SCHEDULER_AUTOTRADE_LIMIT must be >= 2")
	}
	if cfg.AutoTradeQty <= 0 {
		return fmt.Errorf("SCHEDULER_AUTOTRADE_QTY must be > 0")
	}
	switch cfg.AutoTradeStrategy {
	case "sma", "sma_crossover", "infinite_buy":
	default:
		return fmt.Errorf("unsupported SCHEDULER_AUTOTRADE_STRATEGY: %s", cfg.AutoTradeStrategy)
	}
	if cfg.SMALongWindow <= cfg.SMAShortWindow || cfg.SMAShortWindow < 2 {
		return fmt.Errorf("SCHEDULER_SMA_SHORT/LONG are invalid")
	}
	if cfg.InfiniteBuyCount < 0 || cfg.InfiniteBuyCount > 40 {
		return fmt.Errorf("SCHEDULER_INFINITE_BUY_COUNT must be between 0 and 40")
	}
	if cfg.InfiniteBuyCount > 0 && cfg.InfiniteAvgPrice <= 0 {
		return fmt.Errorf("SCHEDULER_INFINITE_AVG_PRICE must be > 0 when buy count > 0")
	}
	if cfg.InfiniteAvgPrice < 0 {
		return fmt.Errorf("SCHEDULER_INFINITE_AVG_PRICE must be >= 0")
	}
	if cfg.InfiniteAllocation < 0 {
		return fmt.Errorf("SCHEDULER_INFINITE_ALLOCATION must be >= 0")
	}
	return nil
}
