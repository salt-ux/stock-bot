package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/salt-ux/stock-bot/internal/auth"
	"github.com/salt-ux/stock-bot/internal/board"
	"github.com/salt-ux/stock-bot/internal/broker"
	"github.com/salt-ux/stock-bot/internal/brokers"
	"github.com/salt-ux/stock-bot/internal/config"
	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/market/kiwoom"
	"github.com/salt-ux/stock-bot/internal/markets"
	"github.com/salt-ux/stock-bot/internal/scheduler"
	"github.com/salt-ux/stock-bot/internal/strategy"
	"github.com/salt-ux/stock-bot/internal/strategy/infinitebuy"
	"github.com/salt-ux/stock-bot/internal/strategy/sma"
	"github.com/salt-ux/stock-bot/internal/trading"
)

type healthResponse struct {
	Status string `json:"status"`
}

const (
	authSessionCookieName          = "stockbot_session"
	authSessionCookieValue         = "active"
	authSessionCookieMaxAgeSeconds = 60 * 60 * 24
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	validator, err := brokers.NewCredentialValidator(cfg)
	if err != nil {
		log.Fatalf("failed to initialize broker validator: %v", err)
	}
	authStore, err := auth.NewMySQLStore(cfg.DB)
	if err != nil {
		log.Fatalf("failed to initialize auth store: %v", err)
	}
	symbolBoardStore, err := board.NewMySQLSymbolStore(cfg.DB)
	if err != nil {
		log.Fatalf("failed to initialize symbol board store: %v", err)
	}
	marketSvc, err := markets.NewService(cfg)
	if err != nil {
		log.Fatalf("failed to initialize market service: %v", err)
	}
	paperSvc, err := trading.NewService(marketSvc, trading.Config{
		InitialCash:      cfg.Trading.InitialCash,
		DuplicateWindow:  time.Duration(cfg.Trading.DuplicateWindowSeconds) * time.Second,
		MaxRecentOrders:  200,
		RiskMaxNotional:  cfg.Trading.MaxPositionNotional,
		RiskDailyLossCap: cfg.Trading.DailyLossLimit,
	})
	if err != nil {
		log.Fatalf("failed to initialize trading service: %v", err)
	}
	buyRuleExecutor := trading.NewBuyRuleExecutor(paperSvc, marketSvc)
	v22SimulationRunner := trading.NewV22SimulationRunner(marketSvc, paperSvc, buyRuleExecutor)
	strategyEngine := strategy.NewEngine(marketSvc)
	eventsHub := newSSEEventsHub()
	v22SimulationRunner.SetHooks(func(tick trading.V22SimulationTick) {
		eventsHub.Publish("v22_sim_tick", tick)
		eventsHub.Publish("paper_state", paperSvc.GetState())
	}, func(state trading.V22SimulationState) {
		eventsHub.Publish("v22_sim_state", state)
		eventsHub.Publish("paper_state", paperSvc.GetState())
	})
	schedulerRunner, err := scheduler.NewCronSchedulerRunner(cfg.Scheduler, strategyEngine, paperSvc, symbolBoardStore, buyRuleExecutor)
	if err != nil {
		log.Fatalf("failed to initialize scheduler: %v", err)
	}
	schedulerRunner.SetAutoTradeListener(func(result scheduler.AutoTradeJobResult) {
		eventsHub.Publish("auto_trade", result)
		eventsHub.Publish("paper_state", paperSvc.GetState())
	})
	schedulerRunner.Start()
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := schedulerRunner.Stop(stopCtx); err != nil {
			log.Printf("failed to stop scheduler: %v", err)
		}
		if err := marketSvc.Close(); err != nil {
			log.Printf("failed to close market service: %v", err)
		}
		if err := authStore.Close(); err != nil {
			log.Printf("failed to close auth store: %v", err)
		}
		if err := symbolBoardStore.Close(); err != nil {
			log.Printf("failed to close symbol board store: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/broker/validate", brokerValidateHandler(validator))
	mux.HandleFunc("/kiwoom/validate", brokerValidateHandler(validator))
	baseSymbolResolver := market.NewDefaultSymbolLookupResolver()
	symbolResolver := baseSymbolResolver
	if strings.EqualFold(strings.TrimSpace(cfg.Market.Provider), "kiwoom") {
		symbolSearcher := kiwoom.NewSymbolLookupSearcher(cfg.Kiwoom, cfg.Market)
		symbolResolver = market.NewRemoteFallbackSymbolLookupResolver(
			symbolSearcher,
			baseSymbolResolver,
			4*time.Second,
		)
	}
	mux.HandleFunc("/market/symbols/search", marketSymbolSearchHandler(symbolResolver))
	mux.HandleFunc("/market/quote", marketQuoteHandler(marketSvc, symbolResolver))
	mux.HandleFunc("/market/candles", marketCandlesHandler(marketSvc, symbolResolver))
	mux.HandleFunc("/board/symbols", boardSymbolsHandler(symbolBoardStore))
	mux.HandleFunc("/paper/order", paperOrderHandler(paperSvc, eventsHub))
	mux.HandleFunc("/paper/state", paperStateHandler(paperSvc))
	mux.HandleFunc("/paper/buyrule/execute", paperBuyRuleExecuteHandler(buyRuleExecutor, paperSvc, eventsHub))
	mux.HandleFunc("/paper/v22-sim/start", paperV22SimulationStartHandler(v22SimulationRunner))
	mux.HandleFunc("/paper/v22-sim/state", paperV22SimulationStateHandler(v22SimulationRunner))
	mux.HandleFunc("/paper/v22-sim/stop", paperV22SimulationStopHandler(v22SimulationRunner))
	mux.HandleFunc("/strategy/list", strategyListHandler)
	mux.HandleFunc("/strategy/signal", strategySignalHandler(strategyEngine))
	mux.HandleFunc("/scheduler/status", schedulerStatusHandler(schedulerRunner))
	mux.HandleFunc("/events/stream", sseEventsHandler(eventsHub))
	mux.HandleFunc("/auth/register", registerHandler(authStore))
	mux.HandleFunc("/auth/login", loginHandler(authStore))
	mux.HandleFunc("/auth/logout", logoutHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/login", loginPageHandler)
	mux.HandleFunc("/trade", tradePageHandler)
	mux.HandleFunc("/", rootHandler)

	addr := cfg.App.Address()
	log.Printf("api server started on %s", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go func() {
		<-shutdownSignal()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}

func shutdownSignal() <-chan os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	return sigCh
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}

func brokerValidateHandler(validator broker.CredentialValidator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		result := validator.ValidateCredentials(ctx)

		w.Header().Set("Content-Type", "application/json")
		if result.Valid {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if isAuthenticated(r) {
		http.Redirect(w, r, "/trade", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if isAuthenticated(r) {
		http.Redirect(w, r, "/trade", http.StatusFound)
		return
	}

	pagePath := filepath.Join("web", "login.html")
	http.ServeFile(w, r, pagePath)
}

func tradePageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	pagePath := filepath.Join("web", "trade.html")
	http.ServeFile(w, r, pagePath)
}

type loginRequest struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

type loginResponse struct {
	Message    string `json:"message"`
	RedirectTo string `json:"redirect_to,omitempty"`
}

func registerHandler(store auth.CredentialsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req loginRequest
		if err := decodeJSON(r.Body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, loginResponse{Message: "요청 형식이 올바르지 않습니다"})
			return
		}
		if err := store.Register(req.ID, req.Password); err != nil {
			switch {
			case errors.Is(err, auth.ErrIDRequired):
				writeJSON(w, http.StatusBadRequest, loginResponse{Message: "아이디를 입력해 주세요"})
			case errors.Is(err, auth.ErrPasswordTooShort):
				writeJSON(w, http.StatusBadRequest, loginResponse{Message: "비밀번호는 5글자 이상이어야 합니다"})
			case errors.Is(err, auth.ErrUserAlreadyExists):
				writeJSON(w, http.StatusConflict, loginResponse{Message: "이미 계정이 등록되어 있습니다"})
			default:
				writeJSON(w, http.StatusInternalServerError, loginResponse{Message: "회원가입 처리 중 오류가 발생했습니다"})
			}
			return
		}

		writeJSON(w, http.StatusCreated, loginResponse{Message: "회원가입이 완료되었습니다"})
	}
}

func loginHandler(store auth.CredentialsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req loginRequest
		if err := decodeJSON(r.Body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, loginResponse{Message: "요청 형식이 올바르지 않습니다"})
			return
		}
		if strings.TrimSpace(req.ID) == "" || strings.TrimSpace(req.Password) == "" {
			writeJSON(w, http.StatusBadRequest, loginResponse{Message: "아이디와 비밀번호를 입력해 주세요"})
			return
		}

		if err := store.Authenticate(req.ID, req.Password); err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				writeJSON(w, http.StatusUnauthorized, loginResponse{Message: "아이디 또는 비밀번호가 올바르지 않습니다"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, loginResponse{Message: "로그인 처리 중 오류가 발생했습니다"})
			return
		}
		setAuthSessionCookie(w)

		writeJSON(w, http.StatusOK, loginResponse{
			Message:    "로그인 성공",
			RedirectTo: "/",
		})
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clearAuthSessionCookie(w)
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "로그아웃되었습니다"})
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(authSessionCookieName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(cookie.Value) == authSessionCookieValue
}

func setAuthSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    authSessionCookieValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   authSessionCookieMaxAgeSeconds,
	})
}

func clearAuthSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func decodeJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("multiple JSON values are not allowed")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func marketQuoteHandler(svc *market.Service, resolver market.SymbolLookupResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw := strings.TrimSpace(r.URL.Query().Get("symbol"))
		if raw == "" {
			raw = strings.TrimSpace(r.URL.Query().Get("query"))
		}
		if raw == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "symbol query is required"})
			return
		}
		symbol, err := resolveSymbolQuery(raw, resolver)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}

		quote, err := svc.Quote(r.Context(), symbol)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, quote)
	}
}

func marketCandlesHandler(svc *market.Service, resolver market.SymbolLookupResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw := strings.TrimSpace(r.URL.Query().Get("symbol"))
		if raw == "" {
			raw = strings.TrimSpace(r.URL.Query().Get("query"))
		}
		if raw == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "symbol query is required"})
			return
		}
		symbol, err := resolveSymbolQuery(raw, resolver)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}

		interval := strings.TrimSpace(r.URL.Query().Get("interval"))
		if interval == "" {
			interval = "1d"
		}

		limit := 20
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "limit must be integer"})
				return
			}
			limit = parsed
		}

		candles, err := svc.Candles(r.Context(), symbol, interval, limit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, candles)
	}
}

func marketSymbolSearchHandler(resolver market.SymbolLookupResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("query"))
		if query == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "query is required"})
			return
		}

		limit := parseIntQuery(r, "limit", 10)
		if limit < 1 {
			limit = 10
		}
		if limit > 50 {
			limit = 50
		}

		items := resolver.Search(query, limit)
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func resolveSymbolQuery(raw string, resolver market.SymbolLookupResolver) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("symbol query is required")
	}
	normalized := strings.ToUpper(raw)
	if isSixDigitSymbol(normalized) {
		return normalized, nil
	}
	if resolver == nil {
		return "", errors.New("종목 코드(6자리)를 입력해 주세요")
	}
	item, ok := resolver.Resolve(raw)
	if !ok {
		return "", errors.New("종목명 또는 종목 코드를 찾을 수 없습니다")
	}
	return item.Symbol, nil
}

func isSixDigitSymbol(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func strategySignalHandler(engine *strategy.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
		if symbol == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "symbol query is required"})
			return
		}

		interval := strings.TrimSpace(r.URL.Query().Get("interval"))
		if interval == "" {
			interval = "1d"
		}

		strategyName := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("strategy")))
		if strategyName == "" {
			strategyName = "sma"
		}

		strategyImpl, defaultLimit, err := buildStrategyFromQuery(r, strategyName)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		limit := parseIntQuery(r, "limit", defaultLimit)

		result, err := engine.Run(r.Context(), symbol, interval, limit, strategyImpl)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func strategyListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"strategies": []map[string]any{
			{
				"name":        "sma",
				"description": "simple moving average crossover",
				"params":      []string{"short", "long", "limit"},
			},
			{
				"name":        "infinite_buy",
				"description": "dca 40-split with staged take-profit rules",
				"params":      []string{"buy_count", "avg_price", "allocation", "limit"},
			},
		},
	})
}

func parseFloatQuery(r *http.Request, key string, fallback float64) float64 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

func buildStrategyFromQuery(r *http.Request, strategyName string) (strategy.Strategy, int, error) {
	switch strategyName {
	case "sma", "sma_crossover":
		shortWindow := parseIntQuery(r, "short", 5)
		longWindow := parseIntQuery(r, "long", 20)
		s, err := sma.NewCrossover(shortWindow, longWindow)
		if err != nil {
			return nil, 0, err
		}
		return s, 60, nil
	case "infinite_buy":
		buyCount := parseIntQuery(r, "buy_count", 0)
		avgPrice := parseFloatQuery(r, "avg_price", 0)
		allocation := parseFloatQuery(r, "allocation", 0)
		s, err := infinitebuy.New(buyCount, avgPrice, allocation)
		if err != nil {
			return nil, 0, err
		}
		return s, 2, nil
	default:
		return nil, 0, errors.New("unsupported strategy; use sma or infinite_buy")
	}
}

func paperOrderHandler(svc *trading.Service, eventsHub *sseEventsHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req trading.OrderRequest
		if err := decodeJSON(r.Body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "요청 형식이 올바르지 않습니다"})
			return
		}

		order, err := svc.PlaceOrder(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		if eventsHub != nil {
			eventsHub.Publish("paper_order", order)
			eventsHub.Publish("paper_state", svc.GetState())
		}
		writeJSON(w, http.StatusOK, order)
	}
}

func paperStateHandler(svc *trading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, svc.GetState())
	}
}

func schedulerStatusHandler(schedulerRunner *scheduler.CronSchedulerRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, schedulerRunner.Snapshot())
	}
}
