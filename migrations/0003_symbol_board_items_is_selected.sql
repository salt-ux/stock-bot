ALTER TABLE symbol_board_items
  ADD COLUMN IF NOT EXISTS is_selected TINYINT(1) NOT NULL DEFAULT 0 AFTER split_count;
