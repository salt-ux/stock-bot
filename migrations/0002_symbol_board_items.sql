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
