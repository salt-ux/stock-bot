package board

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"

	"github.com/salt-ux/stock-bot/internal/config"
)

const (
	maxPrincipalKRW int64 = 1000000000000
)

type MySQLSymbolStore struct {
	db *sql.DB
}

func NewMySQLSymbolStore(cfg config.DBConfig) (*MySQLSymbolStore, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	store := &MySQLSymbolStore{db: db}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *MySQLSymbolStore) Close() error {
	return s.db.Close()
}

func (s *MySQLSymbolStore) List(ctx context.Context) ([]SymbolRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT symbol, display_name, principal_krw, split_count, is_selected, progress_state, sell_ratio_pct, trade_method, note_text, sort_order
FROM symbol_board_items
ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list symbol board: %w", err)
	}
	defer rows.Close()

	items := make([]SymbolRecord, 0, 32)
	for rows.Next() {
		var item SymbolRecord
		if err := rows.Scan(&item.Symbol, &item.DisplayName, &item.PrincipalKRW, &item.SplitCount, &item.IsSelected, &item.ProgressState, &item.SellRatioPct, &item.TradeMethod, &item.NoteText, &item.SortOrder); err != nil {
			return nil, fmt.Errorf("scan symbol board row: %w", err)
		}
		normalized, err := normalizeSymbolRecord(item, len(items))
		if err != nil {
			continue
		}
		items = append(items, normalized)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate symbol board rows: %w", err)
	}

	return items, nil
}

func (s *MySQLSymbolStore) ReplaceAll(ctx context.Context, items []SymbolRecord) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin symbol board tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM symbol_board_items`); err != nil {
		return fmt.Errorf("clear symbol board: %w", err)
	}

	if len(items) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
INSERT INTO symbol_board_items (symbol, display_name, principal_krw, split_count, is_selected, progress_state, sell_ratio_pct, trade_method, note_text, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare symbol board insert: %w", err)
		}
		defer stmt.Close()

		seen := make(map[string]struct{}, len(items))
		for idx, item := range items {
			normalized, err := normalizeSymbolRecord(item, idx)
			if err != nil {
				return err
			}
			if _, exists := seen[normalized.Symbol]; exists {
				return fmt.Errorf("duplicate symbol: %s", normalized.Symbol)
			}
			seen[normalized.Symbol] = struct{}{}

			if _, err := stmt.ExecContext(
				ctx,
				normalized.Symbol,
				normalized.DisplayName,
				normalized.PrincipalKRW,
				normalized.SplitCount,
				normalized.IsSelected,
				normalized.ProgressState,
				normalized.SellRatioPct,
				normalized.TradeMethod,
				normalized.NoteText,
				idx,
			); err != nil {
				return fmt.Errorf("insert symbol board row: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit symbol board tx: %w", err)
	}
	return nil
}

func (s *MySQLSymbolStore) ensureSchema(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS symbol_board_items (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  symbol VARCHAR(12) NOT NULL,
  display_name VARCHAR(120) NOT NULL,
  principal_krw BIGINT NOT NULL DEFAULT 4000000,
  split_count INT NOT NULL DEFAULT 40,
  is_selected TINYINT(1) NOT NULL DEFAULT 0,
  progress_state VARCHAR(12) NOT NULL DEFAULT 'WAIT',
  sell_ratio_pct INT NOT NULL DEFAULT 10,
  trade_method VARCHAR(32) NOT NULL DEFAULT 'V2.2',
  note_text VARCHAR(255) NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_symbol_board_items_symbol (symbol),
  KEY idx_symbol_board_items_sort_order (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("ensure symbol board schema: %w", err)
	}
	if _, err := s.db.ExecContext(
		ctx,
		`ALTER TABLE symbol_board_items ADD COLUMN is_selected TINYINT(1) NOT NULL DEFAULT 0 AFTER split_count`,
	); err != nil && !isMySQLDuplicateColumnError(err) {
		return fmt.Errorf("ensure symbol board is_selected column: %w", err)
	}
	if _, err := s.db.ExecContext(
		ctx,
		`ALTER TABLE symbol_board_items ADD COLUMN progress_state VARCHAR(12) NOT NULL DEFAULT 'WAIT' AFTER is_selected`,
	); err != nil && !isMySQLDuplicateColumnError(err) {
		return fmt.Errorf("ensure symbol board progress_state column: %w", err)
	}
	if _, err := s.db.ExecContext(
		ctx,
		`ALTER TABLE symbol_board_items ADD COLUMN sell_ratio_pct INT NOT NULL DEFAULT 10 AFTER progress_state`,
	); err != nil && !isMySQLDuplicateColumnError(err) {
		return fmt.Errorf("ensure symbol board sell_ratio_pct column: %w", err)
	}
	if _, err := s.db.ExecContext(
		ctx,
		`ALTER TABLE symbol_board_items ADD COLUMN trade_method VARCHAR(32) NOT NULL DEFAULT 'V2.2' AFTER sell_ratio_pct`,
	); err != nil && !isMySQLDuplicateColumnError(err) {
		return fmt.Errorf("ensure symbol board trade_method column: %w", err)
	}
	if _, err := s.db.ExecContext(
		ctx,
		`ALTER TABLE symbol_board_items ADD COLUMN note_text VARCHAR(255) NOT NULL DEFAULT '' AFTER trade_method`,
	); err != nil && !isMySQLDuplicateColumnError(err) {
		return fmt.Errorf("ensure symbol board note_text column: %w", err)
	}
	return nil
}

func normalizeSymbolRecord(item SymbolRecord, sortOrder int) (SymbolRecord, error) {
	symbol := normalizeSymbol(item.Symbol)
	if !isValidSymbol(symbol) {
		return SymbolRecord{}, fmt.Errorf("invalid symbol: %q", item.Symbol)
	}

	displayName := strings.TrimSpace(item.DisplayName)
	if displayName == "" {
		displayName = symbol
	}
	runes := []rune(displayName)
	if len(runes) > 120 {
		displayName = string(runes[:120])
		displayName = strings.TrimSpace(displayName)
		if displayName == "" {
			displayName = symbol
		}
	}

	principal := normalizePrincipal(item.PrincipalKRW)
	splitCount := normalizeSplit(item.SplitCount)
	progressState := normalizeProgressState(item.ProgressState)
	sellRatioPct := normalizeSellRatioPct(item.SellRatioPct)
	tradeMethod := normalizeTradeMethod(item.TradeMethod)
	noteText := normalizeNoteText(item.NoteText)
	if sortOrder < 0 {
		sortOrder = 0
	}

	return SymbolRecord{
		Symbol:        symbol,
		DisplayName:   displayName,
		PrincipalKRW:  principal,
		SplitCount:    splitCount,
		IsSelected:    item.IsSelected,
		ProgressState: progressState,
		SellRatioPct:  sellRatioPct,
		TradeMethod:   tradeMethod,
		NoteText:      noteText,
		SortOrder:     sortOrder,
	}, nil
}

func isMySQLDuplicateColumnError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	return mysqlErr.Number == 1060
}

func normalizeSymbol(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isValidSymbol(symbol string) bool {
	if len(symbol) < 1 || len(symbol) > 12 {
		return false
	}
	for _, r := range symbol {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func normalizePrincipal(value int64) int64 {
	if value < 1 {
		return DefaultPrincipalKRW
	}
	if value > maxPrincipalKRW {
		return maxPrincipalKRW
	}
	return value
}

func normalizeSplit(value int) int {
	if value < MinSplitCount {
		return MinSplitCount
	}
	if value > MaxSplitCount {
		return MaxSplitCount
	}
	return value
}

func normalizeProgressState(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case ProgressStateRun:
		return ProgressStateRun
	default:
		return ProgressStateWait
	}
}

func normalizeSellRatioPct(value int) int {
	if value < MinSellRatioPct {
		return MinSellRatioPct
	}
	if value > MaxSellRatioPct {
		return MaxSellRatioPct
	}
	return value
}

func normalizeTradeMethod(value string) string {
	method := strings.TrimSpace(value)
	if method == "" {
		method = DefaultTradeMethod
	}
	runes := []rune(method)
	if len(runes) > 32 {
		method = string(runes[:32])
		method = strings.TrimSpace(method)
		if method == "" {
			method = DefaultTradeMethod
		}
	}
	return method
}

func normalizeNoteText(value string) string {
	note := strings.TrimSpace(value)
	runes := []rune(note)
	if len(runes) > 255 {
		note = string(runes[:255])
		note = strings.TrimSpace(note)
	}
	return note
}
