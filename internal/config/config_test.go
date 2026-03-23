package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("REDIS_HOST", "")
	t.Setenv("REDIS_PORT", "")
	t.Setenv("BROKER_PROVIDER", "")
	t.Setenv("MARKET_PROVIDER", "")
	t.Setenv("MARKET_CACHE_TTL_SECONDS", "")
	t.Setenv("TRADING_INITIAL_CASH", "")
	t.Setenv("RISK_MAX_POSITION_NOTIONAL", "")
	t.Setenv("RISK_DAILY_LOSS_LIMIT", "")
	t.Setenv("RISK_DUPLICATE_WINDOW_SECONDS", "")
	t.Setenv("SCHEDULER_ENABLED", "")
	t.Setenv("SCHEDULER_TIMEZONE", "")
	t.Setenv("SCHEDULER_MARKET_OPEN_CRON", "")
	t.Setenv("SCHEDULER_AUTOTRADE_CRON", "")
	t.Setenv("SCHEDULER_AUTOTRADE_SYMBOL", "")
	t.Setenv("SCHEDULER_AUTOTRADE_INTERVAL", "")
	t.Setenv("SCHEDULER_AUTOTRADE_LIMIT", "")
	t.Setenv("SCHEDULER_AUTOTRADE_STRATEGY", "")
	t.Setenv("SCHEDULER_AUTOTRADE_QTY", "")
	t.Setenv("SCHEDULER_SMA_SHORT", "")
	t.Setenv("SCHEDULER_SMA_LONG", "")
	t.Setenv("SCHEDULER_INFINITE_BUY_COUNT", "")
	t.Setenv("SCHEDULER_INFINITE_AVG_PRICE", "")
	t.Setenv("SCHEDULER_INFINITE_ALLOCATION", "")
	t.Setenv("KIWOOM_MARKET_QUOTE_PATH", "")
	t.Setenv("KIWOOM_MARKET_QUOTE_API_ID", "")
	t.Setenv("KIWOOM_MARKET_CANDLE_PATH", "")
	t.Setenv("KIWOOM_MARKET_CANDLE_API_ID", "")
	t.Setenv("KIWOOM_MARKET_CANDLE_MINUTE_API_ID", "")
	t.Setenv("KIWOOM_MARKET_CANDLE_DAILY_API_ID", "")
	t.Setenv("KIWOOM_MARKET_SYMBOL_PATH", "")
	t.Setenv("KIWOOM_MARKET_SYMBOL_LIST_API_ID", "")
	t.Setenv("KIWOOM_MARKET_SYMBOL_SEARCH_API_ID", "")
	t.Setenv("KIWOOM_MARKET_SYMBOL_CACHE_TTL_SECONDS", "")
	t.Setenv("KIWOOM_MARKET_USE_WEBSOCKET", "")
	t.Setenv("KIWOOM_MARKET_WS_URL", "")
	t.Setenv("KIWOOM_MARKET_WS_PATH", "")
	t.Setenv("KIWOOM_MARKET_WS_QUOTE_TR", "")
	t.Setenv("KIWOOM_BASE_URL", "")
	t.Setenv("KIWOOM_TOKEN_PATH", "")
	t.Setenv("KIWOOM_APP_KEY", "")
	t.Setenv("KIWOOM_APP_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Env != "local" {
		t.Fatalf("expected local env, got %s", cfg.App.Env)
	}
	if cfg.App.Port != 8080 {
		t.Fatalf("expected app port 8080, got %d", cfg.App.Port)
	}
	if cfg.DB.Port != 3306 {
		t.Fatalf("expected db port 3306, got %d", cfg.DB.Port)
	}
	if cfg.Redis.Port != 6379 {
		t.Fatalf("expected redis port 6379, got %d", cfg.Redis.Port)
	}
	if cfg.Kiwoom.TokenPath != "/oauth2/token" {
		t.Fatalf("expected kiwoom token path /oauth2/token, got %s", cfg.Kiwoom.TokenPath)
	}
	if cfg.Broker.Provider != "kiwoom" {
		t.Fatalf("expected broker provider kiwoom, got %s", cfg.Broker.Provider)
	}
	if cfg.Market.Provider != "mock" {
		t.Fatalf("expected market provider mock, got %s", cfg.Market.Provider)
	}
	if cfg.Market.CacheTTLSeconds != 5 {
		t.Fatalf("expected market cache ttl 5, got %d", cfg.Market.CacheTTLSeconds)
	}
	if cfg.Market.QuotePath != "/api/dostk/stkinfo" {
		t.Fatalf("unexpected default quote path: %s", cfg.Market.QuotePath)
	}
	if cfg.Market.CandlePath != "/api/dostk/chart" {
		t.Fatalf("unexpected default candle path: %s", cfg.Market.CandlePath)
	}
	if cfg.Market.CandleMinuteAPIID != "ka10080" {
		t.Fatalf("unexpected default minute candle api id: %s", cfg.Market.CandleMinuteAPIID)
	}
	if cfg.Market.CandleDailyAPIID != "ka10081" {
		t.Fatalf("unexpected default daily candle api id: %s", cfg.Market.CandleDailyAPIID)
	}
	if cfg.Market.SymbolPath != "/api/dostk/stkinfo" {
		t.Fatalf("unexpected default symbol path: %s", cfg.Market.SymbolPath)
	}
	if cfg.Market.SymbolListAPIID != "ka10099" {
		t.Fatalf("unexpected default symbol list api id: %s", cfg.Market.SymbolListAPIID)
	}
	if cfg.Market.SymbolSearchAPIID != "ka10100" {
		t.Fatalf("unexpected default symbol search api id: %s", cfg.Market.SymbolSearchAPIID)
	}
	if cfg.Market.SymbolCacheTTL != 600 {
		t.Fatalf("unexpected default symbol cache ttl: %d", cfg.Market.SymbolCacheTTL)
	}
	if !cfg.Market.UseWebSocket {
		t.Fatalf("expected websocket enabled by default")
	}
	if cfg.Market.WebSocketPath != "/api/dostk/websocket" {
		t.Fatalf("unexpected ws path: %s", cfg.Market.WebSocketPath)
	}
	if cfg.Market.WebSocketQuoteTR != "0B" {
		t.Fatalf("unexpected ws tr: %s", cfg.Market.WebSocketQuoteTR)
	}
	if cfg.Trading.InitialCash != 50000000 {
		t.Fatalf("unexpected default initial cash: %f", cfg.Trading.InitialCash)
	}
	if cfg.Trading.MaxPositionNotional != 10000000 {
		t.Fatalf("unexpected default max position notional: %f", cfg.Trading.MaxPositionNotional)
	}
	if cfg.Trading.DailyLossLimit != 1000000 {
		t.Fatalf("unexpected default daily loss limit: %f", cfg.Trading.DailyLossLimit)
	}
	if cfg.Trading.DuplicateWindowSeconds != 30 {
		t.Fatalf("unexpected default duplicate window: %d", cfg.Trading.DuplicateWindowSeconds)
	}
	if cfg.Scheduler.Enabled {
		t.Fatal("expected scheduler to be disabled by default")
	}
	if cfg.Scheduler.Timezone != "Asia/Seoul" {
		t.Fatalf("unexpected scheduler timezone: %s", cfg.Scheduler.Timezone)
	}
}

func TestLoadInvalidPortRange(t *testing.T) {
	t.Setenv("APP_PORT", "70000")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for out-of-range APP_PORT")
	}
}

func TestLoadKiwoomPartialConfig(t *testing.T) {
	t.Setenv("BROKER_PROVIDER", "kiwoom")
	t.Setenv("KIWOOM_APP_KEY", "my-app-key")
	t.Setenv("KIWOOM_APP_SECRET", "my-app-secret")
	t.Setenv("KIWOOM_BASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when KIWOOM_BASE_URL is missing")
	}
}

func TestLoadUnsupportedBroker(t *testing.T) {
	t.Setenv("BROKER_PROVIDER", "unknown")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unsupported broker provider")
	}
}

func TestLoadUnsupportedMarketProvider(t *testing.T) {
	t.Setenv("MARKET_PROVIDER", "unknown")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unsupported market provider")
	}
}

func TestLoadKiwoomMarketRequiresKiwoomConfig(t *testing.T) {
	t.Setenv("MARKET_PROVIDER", "kiwoom")
	t.Setenv("KIWOOM_BASE_URL", "")
	t.Setenv("KIWOOM_APP_KEY", "")
	t.Setenv("KIWOOM_APP_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when kiwoom market provider has no kiwoom config")
	}
}

func TestLoadInvalidTradingConfig(t *testing.T) {
	t.Setenv("TRADING_INITIAL_CASH", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid trading config")
	}
}

func TestLoadInvalidSchedulerConfig(t *testing.T) {
	t.Setenv("SCHEDULER_ENABLED", "true")
	t.Setenv("SCHEDULER_AUTOTRADE_QTY", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid scheduler config")
	}
}

func TestLoadKiwoomAliasKeys(t *testing.T) {
	t.Setenv("KIWOOM_APP_KEY", "")
	t.Setenv("KIWOOM_APP_SECRET", "")
	t.Setenv("APP_KEY", "alias-key-123")
	t.Setenv("APP_SEC", "alias-sec-123")
	t.Setenv("KIWOOM_BASE_URL", "https://example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Kiwoom.AppKey != "alias-key-123" {
		t.Fatalf("expected APP_KEY alias to be used")
	}
	if cfg.Kiwoom.AppSecret != "alias-sec-123" {
		t.Fatalf("expected APP_SEC alias to be used")
	}
}
