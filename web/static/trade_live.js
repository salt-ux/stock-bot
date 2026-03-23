const sseStatus = document.getElementById("sse-status");
const lastEvent = document.getElementById("last-event");
const cashSummary = document.getElementById("cash-summary");
const positionsBody = document.getElementById("positions-body");
const ordersBody = document.getElementById("orders-body");
const orderForm = document.getElementById("paper-order-form");
const orderMessage = document.getElementById("order-message");
const quoteForm = document.getElementById("quote-form");
const quoteSymbolCodeInput = document.getElementById("quote-symbol-code");
const quoteSymbolNameInput = document.getElementById("quote-symbol-name");
const quoteSymbolNameOptions = document.getElementById("quote-symbol-name-options");
const quoteMessage = document.getElementById("quote-message");
const quoteResult = document.getElementById("quote-result");
const rangeButtons = document.getElementById("range-buttons");
const candleChart = document.getElementById("candle-chart");
const accountCash = document.getElementById("account-cash");
const accountHoldings = document.getElementById("account-holdings");
const accountTotal = document.getElementById("account-total");
const accountRealizedPnL = document.getElementById("account-realized-pnl");
const accountPositionCount = document.getElementById("account-position-count");
const accountOrderCount = document.getElementById("account-order-count");
const accountUpdatedAt = document.getElementById("account-updated-at");
const strategyGridBody = document.getElementById("strategy-grid-body");
const symbolManageForm = document.getElementById("symbol-manage-form");
const symbolManageCodeInput = document.getElementById("symbol-manage-code-input");
const symbolManageNameInput = document.getElementById("symbol-manage-name-input");
const symbolManageNameSearchButton = document.getElementById("symbol-manage-name-search-button");
const symbolManageNameSuggestions = document.getElementById("symbol-manage-name-suggestions");
const symbolManageMessage = document.getElementById("symbol-manage-message");
const symbolRunSelectedRuleButton = document.getElementById("symbol-run-selected-rule-button");
const symbolRemoveSelectedButton = document.getElementById("symbol-remove-selected-button");
const symbolResetButton = document.getElementById("symbol-reset-button");
const v22SimStartButton = document.getElementById("v22-sim-start-button");
const v22SimStopButton = document.getElementById("v22-sim-stop-button");
const v22SimMessage = document.getElementById("v22-sim-message");
const v22SimLogBody = document.getElementById("v22-sim-log-body");

let selectedRange = "1d";
let quoteNameSearchDebounceTimer = null;
let symbolManageNameSearchDebounceTimer = null;
let symbolManageNameSuggestionItems = [];
let symbolManageNameSuggestionIndex = -1;
const STRATEGY_UNIVERSE_STORAGE_KEY = "mock_hts_strategy_universe_v1";
const SYMBOL_CONFIG_STORAGE_KEY = "mock_hts_symbol_config_v1";
const SYMBOL_BOARD_API_ENDPOINT = "/board/symbols";
const V22_SIM_START_API_ENDPOINT = "/paper/v22-sim/start";
const V22_SIM_STATE_API_ENDPOINT = "/paper/v22-sim/state";
const V22_SIM_STOP_API_ENDPOINT = "/paper/v22-sim/stop";
const DEFAULT_PRINCIPAL_KRW = 4000000;
const DEFAULT_SPLIT_COUNT = 40;
const MIN_SPLIT_COUNT = 10;
const MAX_SPLIT_COUNT = 50;
const MAX_PRINCIPAL_KRW = 1000000000000;
const PROGRESS_STATE_WAIT = "WAIT";
const PROGRESS_STATE_RUN = "RUN";
const DEFAULT_PROGRESS_STATE = PROGRESS_STATE_WAIT;
const DEFAULT_SELL_RATIO_PCT = 10;
const MIN_SELL_RATIO_PCT = 0;
const MAX_SELL_RATIO_PCT = 100;
const DEFAULT_TRADE_METHOD = "V2.2";
const GRID_QUOTE_REFRESH_INTERVAL_MS = 20000;
const DEFAULT_STRATEGY_UNIVERSE = [
  "AMCR", "BAC", "BEN", "CMCSA", "CSCO",
  "GIS", "HRL", "INTC", "KEY", "KMI",
  "KO", "KR", "MDT", "NEE", "PFE",
  "PPL", "T", "TFC", "VTRS", "VZ",
];
let strategyUniverse = [...DEFAULT_STRATEGY_UNIVERSE];
let symbolConfigByCode = {};
let gridQuotePriceBySymbol = {};
let gridQuoteRefreshTimer = null;
let gridQuoteRefreshInFlight = false;
let v22SimStateSnapshot = null;
let latestStateSnapshot = {
  cash: 0,
  realized_pnl_today: 0,
  positions: [],
  recent_orders: [],
};

function won(v) {
  return Number(v || 0).toLocaleString("ko-KR", { maximumFractionDigits: 2 });
}

function setSSEStatus(text, cls) {
  sseStatus.textContent = text;
  sseStatus.className = `badge ${cls || ""}`.trim();
}

function setLastEvent(text) {
  lastEvent.textContent = text;
}

function formatDateTime(value) {
  const d = value ? new Date(value) : new Date();
  if (!Number.isFinite(d.getTime())) {
    return "-";
  }

  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Seoul",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
  const parts = formatter.formatToParts(d);
  const map = {};
  parts.forEach((p) => {
    if (p.type !== "literal") {
      map[p.type] = p.value;
    }
  });
  if (!map.year || !map.month || !map.day || !map.hour || !map.minute || !map.second) {
    return "-";
  }
  return `${map.year}-${map.month}-${map.day} ${map.hour}:${map.minute}:${map.second}`;
}

function formatKSTNow() {
  return formatDateTime(new Date());
}

function normalizeBoardSymbol(value) {
  return String(value || "")
    .trim()
    .toUpperCase()
    .replace(/\s+/g, "");
}

function extractBoardSymbolCandidate(value) {
  const raw = String(value || "").trim();
  if (raw === "") {
    return "";
  }
  const symbolPart = raw.includes(",") ? raw.split(",")[0] : raw;
  return normalizeBoardSymbol(symbolPart);
}

function extractInlineBoardName(value) {
  const raw = String(value || "");
  const idx = raw.indexOf(",");
  if (idx < 0) {
    return "";
  }
  return String(raw.slice(idx + 1)).trim();
}

function isValidBoardSymbol(symbol) {
  return /^[A-Z0-9._-]{1,12}$/.test(symbol);
}

function normalizeBoardSymbolName(value, symbolFallback = "") {
  const trimmed = String(value || "").trim();
  if (trimmed !== "") {
    return trimmed;
  }
  return String(symbolFallback || "").trim();
}

function normalizePrincipalValue(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return DEFAULT_PRINCIPAL_KRW;
  }
  const rounded = Math.floor(parsed);
  if (rounded < 1) {
    return DEFAULT_PRINCIPAL_KRW;
  }
  if (rounded > MAX_PRINCIPAL_KRW) {
    return MAX_PRINCIPAL_KRW;
  }
  return rounded;
}

function normalizeSplitValue(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return DEFAULT_SPLIT_COUNT;
  }
  const rounded = Math.floor(parsed);
  if (rounded < MIN_SPLIT_COUNT) {
    return MIN_SPLIT_COUNT;
  }
  if (rounded > MAX_SPLIT_COUNT) {
    return MAX_SPLIT_COUNT;
  }
  return rounded;
}

function normalizeSelectedValue(value) {
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "number") {
    return value !== 0;
  }
  const text = String(value || "").trim().toLowerCase();
  return text === "1" || text === "true" || text === "y" || text === "yes";
}

function normalizeProgressState(value) {
  const text = String(value || "").trim().toUpperCase();
  return text === PROGRESS_STATE_RUN ? PROGRESS_STATE_RUN : PROGRESS_STATE_WAIT;
}

function normalizeSellRatioValue(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return DEFAULT_SELL_RATIO_PCT;
  }
  const rounded = Math.floor(parsed);
  if (rounded < MIN_SELL_RATIO_PCT) {
    return MIN_SELL_RATIO_PCT;
  }
  if (rounded > MAX_SELL_RATIO_PCT) {
    return MAX_SELL_RATIO_PCT;
  }
  return rounded;
}

function normalizeTradeMethodValue(value) {
  const trimmed = String(value || "").trim();
  if (trimmed === "") {
    return DEFAULT_TRADE_METHOD;
  }
  return trimmed.slice(0, 32);
}

function normalizeNoteTextValue(value) {
  return String(value || "").trim().slice(0, 255);
}

function setSymbolManageMessage(text, tone = "") {
  if (!symbolManageMessage) {
    return;
  }
  symbolManageMessage.className = "line-status";
  if (tone === "ok") {
    symbolManageMessage.classList.add("profit");
  }
  if (tone === "error") {
    symbolManageMessage.classList.add("loss");
  }
  symbolManageMessage.textContent = text;
}

function setV22SimMessage(text, tone = "") {
  if (!v22SimMessage) {
    return;
  }
  v22SimMessage.className = "line-status";
  if (tone === "ok") {
    v22SimMessage.classList.add("profit");
  }
  if (tone === "error") {
    v22SimMessage.classList.add("loss");
  }
  v22SimMessage.textContent = text;
}

function formatV22SimOrderSummary(tick) {
  const total = Number(tick?.total_orders || 0);
  const buy = Number(tick?.total_buy_orders || 0);
  const sell = Number(tick?.total_sell_orders || 0);
  return `${total}건 (매수 ${buy}, 매도 ${sell})`;
}

function renderV22SimLog(state) {
  if (!v22SimLogBody) {
    return;
  }
  const ticks = Array.isArray(state?.ticks) ? state.ticks : [];
  if (ticks.length === 0) {
    v22SimLogBody.innerHTML = `<tr><td colspan="5" class="cell-note">시뮬레이션 로그가 없습니다.</td></tr>`;
    return;
  }

  const rows = [...ticks]
    .slice(-50)
    .reverse()
    .map((tick) => {
      const day = Number(tick.day || 0);
      const price = Number(tick.price || 0);
      const changePct = Number(tick.change_pct || 0);
      const message = String(tick.message || "-");
      return `
      <tr>
        <td>${day}</td>
        <td>${won(price)}</td>
        <td>${changePct.toFixed(2)}%</td>
        <td>${formatV22SimOrderSummary(tick)}</td>
        <td>${message}</td>
      </tr>`;
    })
    .join("");
  v22SimLogBody.innerHTML = rows;
}

function renderV22SimState(state) {
  if (!state || typeof state !== "object") {
    return;
  }
  v22SimStateSnapshot = state;
  renderV22SimLog(state);

  const running = Boolean(state.running);
  const symbol = String(state.symbol || "005930");
  const current = Number(state.current_day || 0);
  const days = Number(state.days || 0);
  const err = String(state.error || "");
  if (running) {
    setV22SimMessage(`시뮬레이션 실행중: ${symbol} ${current}/${days}일`, "ok");
  } else if (err !== "" && err !== "stopped") {
    setV22SimMessage(`시뮬레이션 종료(오류): ${err}`, "error");
  } else if (err === "stopped") {
    setV22SimMessage(`시뮬레이션 중지됨: ${symbol} ${current}/${days}일`, "error");
  } else if (days > 0) {
    setV22SimMessage(`시뮬레이션 완료: ${symbol} ${current}/${days}일`, "ok");
  } else {
    setV22SimMessage("시뮬 대기중");
  }
}

function saveStrategyUniverseToStorage() {
  try {
    localStorage.setItem(STRATEGY_UNIVERSE_STORAGE_KEY, JSON.stringify(strategyUniverse));
  } catch (_err) {
    setSymbolManageMessage("종목 목록 저장에 실패했습니다.", "error");
  }
}

function loadStrategyUniverseFromStorage() {
  try {
    const raw = localStorage.getItem(STRATEGY_UNIVERSE_STORAGE_KEY);
    if (!raw) {
      return;
    }
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return;
    }
    const normalized = parsed
      .map((item) => normalizeBoardSymbol(item))
      .filter((item) => isValidBoardSymbol(item));
    strategyUniverse = [...new Set(normalized)];
  } catch (_err) {
    strategyUniverse = [...DEFAULT_STRATEGY_UNIVERSE];
  }
}

function saveSymbolConfigsToStorage() {
  try {
    localStorage.setItem(SYMBOL_CONFIG_STORAGE_KEY, JSON.stringify(symbolConfigByCode));
  } catch (_err) {
    setSymbolManageMessage("종목 설정 저장에 실패했습니다.", "error");
  }
}

function loadSymbolConfigsFromStorage() {
  try {
    const raw = localStorage.getItem(SYMBOL_CONFIG_STORAGE_KEY);
    if (!raw) {
      return;
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return;
    }
    const next = {};
    Object.entries(parsed).forEach(([key, value]) => {
      const symbol = normalizeBoardSymbol(key);
      if (!isValidBoardSymbol(symbol)) {
        return;
      }
      if (!value || typeof value !== "object") {
        return;
      }
      next[symbol] = {
        principal_krw: normalizePrincipalValue(value.principal_krw),
        split_count: normalizeSplitValue(value.split_count),
        display_name: normalizeBoardSymbolName(value.display_name, symbol),
        is_selected: normalizeSelectedValue(value.is_selected),
        progress_state: normalizeProgressState(value.progress_state),
        sell_ratio_pct: normalizeSellRatioValue(value.sell_ratio_pct),
        trade_method: normalizeTradeMethodValue(value.trade_method),
        note_text: normalizeNoteTextValue(value.note_text),
      };
    });
    symbolConfigByCode = next;
  } catch (_err) {
    symbolConfigByCode = {};
  }
}

function syncSymbolConfigsWithUniverse() {
  const next = {};
  strategyUniverse.forEach((symbol) => {
    const current = symbolConfigByCode[symbol] || {};
    next[symbol] = {
      principal_krw: normalizePrincipalValue(current.principal_krw),
      split_count: normalizeSplitValue(current.split_count),
      display_name: normalizeBoardSymbolName(current.display_name, symbol),
      is_selected: normalizeSelectedValue(current.is_selected),
      progress_state: normalizeProgressState(current.progress_state),
      sell_ratio_pct: normalizeSellRatioValue(current.sell_ratio_pct),
      trade_method: normalizeTradeMethodValue(current.trade_method),
      note_text: normalizeNoteTextValue(current.note_text),
    };
  });
  symbolConfigByCode = next;
}

function getSymbolConfig(symbol) {
  const key = normalizeBoardSymbol(symbol);
  const current = symbolConfigByCode[key] || {};
  const normalized = {
    principal_krw: normalizePrincipalValue(current.principal_krw),
    split_count: normalizeSplitValue(current.split_count),
    display_name: normalizeBoardSymbolName(current.display_name, key),
    is_selected: normalizeSelectedValue(current.is_selected),
    progress_state: normalizeProgressState(current.progress_state),
    sell_ratio_pct: normalizeSellRatioValue(current.sell_ratio_pct),
    trade_method: normalizeTradeMethodValue(current.trade_method),
    note_text: normalizeNoteTextValue(current.note_text),
  };
  symbolConfigByCode[key] = normalized;
  return normalized;
}

function calculateConfiguredPrincipalTotal() {
  return strategyUniverse.reduce((sum, symbol) => {
    const cfg = getSymbolConfig(symbol);
    return sum + normalizePrincipalValue(cfg.principal_krw);
  }, 0);
}

function renderConfiguredPrincipalTotal() {
  if (!accountHoldings) {
    return;
  }
  accountHoldings.textContent = won(calculateConfiguredPrincipalTotal());
}

function cloneSymbolConfigMap(source) {
  const next = {};
  Object.entries(source || {}).forEach(([key, value]) => {
    const symbol = normalizeBoardSymbol(key);
    if (!isValidBoardSymbol(symbol)) {
      return;
    }
    next[symbol] = {
      principal_krw: normalizePrincipalValue(value && value.principal_krw),
      split_count: normalizeSplitValue(value && value.split_count),
      display_name: normalizeBoardSymbolName(value && value.display_name, symbol),
      is_selected: normalizeSelectedValue(value && value.is_selected),
      progress_state: normalizeProgressState(value && value.progress_state),
      sell_ratio_pct: normalizeSellRatioValue(value && value.sell_ratio_pct),
      trade_method: normalizeTradeMethodValue(value && value.trade_method),
      note_text: normalizeNoteTextValue(value && value.note_text),
    };
  });
  return next;
}

function normalizeBoardItems(items) {
  if (!Array.isArray(items)) {
    return [];
  }
  const seen = new Set();
  const normalized = [];
  items.forEach((item) => {
    if (!item || typeof item !== "object") {
      return;
    }
    const symbol = normalizeBoardSymbol(item.symbol);
    if (!isValidBoardSymbol(symbol) || seen.has(symbol)) {
      return;
    }
    seen.add(symbol);
    normalized.push({
      symbol,
      display_name: normalizeBoardSymbolName(item.display_name, symbol),
      principal_krw: normalizePrincipalValue(item.principal_krw),
      split_count: normalizeSplitValue(item.split_count),
      is_selected: normalizeSelectedValue(item.is_selected),
      progress_state: normalizeProgressState(item.progress_state),
      sell_ratio_pct: normalizeSellRatioValue(item.sell_ratio_pct),
      trade_method: normalizeTradeMethodValue(item.trade_method),
      note_text: normalizeNoteTextValue(item.note_text),
      sort_order: normalized.length,
    });
  });
  return normalized;
}

function applyBoardItems(items, allowEmpty = false) {
  const normalized = normalizeBoardItems(items);
  if (normalized.length === 0 && !allowEmpty) {
    return false;
  }
  strategyUniverse = normalized.map((item) => item.symbol);
  const next = {};
  normalized.forEach((item) => {
    next[item.symbol] = {
      principal_krw: item.principal_krw,
      split_count: item.split_count,
      display_name: item.display_name,
      is_selected: normalizeSelectedValue(item.is_selected),
      progress_state: normalizeProgressState(item.progress_state),
      sell_ratio_pct: normalizeSellRatioValue(item.sell_ratio_pct),
      trade_method: normalizeTradeMethodValue(item.trade_method),
      note_text: normalizeNoteTextValue(item.note_text),
    };
  });
  symbolConfigByCode = next;
  return true;
}

function toBoardItemsPayload() {
  syncSymbolConfigsWithUniverse();
  return strategyUniverse.map((symbol, idx) => {
    const cfg = getSymbolConfig(symbol);
    return {
      symbol,
      display_name: normalizeBoardSymbolName(cfg.display_name, symbol),
      principal_krw: normalizePrincipalValue(cfg.principal_krw),
      split_count: normalizeSplitValue(cfg.split_count),
      is_selected: normalizeSelectedValue(cfg.is_selected),
      progress_state: normalizeProgressState(cfg.progress_state),
      sell_ratio_pct: normalizeSellRatioValue(cfg.sell_ratio_pct),
      trade_method: normalizeTradeMethodValue(cfg.trade_method),
      note_text: normalizeNoteTextValue(cfg.note_text),
      sort_order: idx,
    };
  });
}

async function fetchBoardItemsFromDB() {
  const res = await fetch(SYMBOL_BOARD_API_ENDPOINT);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.message || "종목 목록을 불러오지 못했습니다.");
  }
  return normalizeBoardItems(body.items);
}

async function saveBoardItemsToDB() {
  const res = await fetch(SYMBOL_BOARD_API_ENDPOINT, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ items: toBoardItemsPayload() }),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.message || "종목 목록 저장에 실패했습니다.");
  }
  if (Array.isArray(body.items)) {
    applyBoardItems(body.items, true);
  }
}

async function persistBoardState() {
  saveStrategyUniverseToStorage();
  saveSymbolConfigsToStorage();
  await saveBoardItemsToDB();
  saveStrategyUniverseToStorage();
  saveSymbolConfigsToStorage();
}

function buildDefaultV22SimulationRequest() {
  const symbol = "005930";
  const cfg = getSymbolConfig(symbol);
  return {
    symbol,
    display_name: "삼성전자",
    days: 50,
    interval_seconds: 10,
    min_change_pct: -3,
    max_change_pct: 10,
    base_price: 70000,
    principal_krw: normalizePrincipalValue(cfg.principal_krw),
    split_count: normalizeSplitValue(cfg.split_count),
  };
}

async function fetchV22SimulationState() {
  const res = await fetch(V22_SIM_STATE_API_ENDPOINT);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.message || "시뮬레이션 상태 조회에 실패했습니다.");
  }
  return body;
}

async function startV22Simulation() {
  setV22SimMessage("시뮬레이션 시작 요청 중...");
  const payload = buildDefaultV22SimulationRequest();
  const res = await fetch(V22_SIM_START_API_ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.message || "시뮬레이션 시작에 실패했습니다.");
  }
  renderV22SimState(body);
}

async function stopV22Simulation() {
  setV22SimMessage("시뮬레이션 중지 요청 중...");
  const res = await fetch(V22_SIM_STOP_API_ENDPOINT, {
    method: "POST",
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.message || "시뮬레이션 중지에 실패했습니다.");
  }
  renderV22SimState(body);
}

async function bootstrapBoardState() {
  let hasLocalUniverse = false;
  try {
    hasLocalUniverse = localStorage.getItem(STRATEGY_UNIVERSE_STORAGE_KEY) !== null;
  } catch (_err) {
    hasLocalUniverse = false;
  }

  loadStrategyUniverseFromStorage();
  loadSymbolConfigsFromStorage();
  syncSymbolConfigsWithUniverse();

  try {
    const remoteItems = await fetchBoardItemsFromDB();
    if (remoteItems.length > 0) {
      applyBoardItems(remoteItems);
      saveStrategyUniverseToStorage();
      saveSymbolConfigsToStorage();
      return;
    }
  } catch (_err) {
    // fallback to local storage/default list below
  }

  if (strategyUniverse.length === 0 && !hasLocalUniverse) {
    strategyUniverse = [...DEFAULT_STRATEGY_UNIVERSE];
    syncSymbolConfigsWithUniverse();
  }
  try {
    await persistBoardState();
  } catch (_err) {
    // keep local state when db save fails
  }
}

function formatSignedPercent(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) {
    return "#N/A";
  }
  const sign = n >= 0 ? "+" : "";
  return `${sign}${n.toFixed(2)}%`;
}

function clearStaleGridQuoteCache() {
  const keep = new Set(strategyUniverse);
  Object.keys(gridQuotePriceBySymbol).forEach((symbol) => {
    if (!keep.has(symbol)) {
      delete gridQuotePriceBySymbol[symbol];
    }
  });
}

function startGridQuoteAutoRefresh() {
  if (gridQuoteRefreshTimer) {
    clearInterval(gridQuoteRefreshTimer);
    gridQuoteRefreshTimer = null;
  }
  if (strategyUniverse.length === 0) {
    return;
  }
  gridQuoteRefreshTimer = setInterval(() => {
    void refreshGridQuotePrices();
  }, GRID_QUOTE_REFRESH_INTERVAL_MS);
}

async function refreshGridQuotePrices() {
  if (gridQuoteRefreshInFlight) {
    return;
  }
  const symbols = [...new Set(strategyUniverse.map((symbol) => normalizeBoardSymbol(symbol)).filter((symbol) => isValidBoardSymbol(symbol)))];
  if (symbols.length === 0) {
    return;
  }

  gridQuoteRefreshInFlight = true;
  try {
    const results = await Promise.allSettled(symbols.map((symbol) => fetchQuote(symbol)));
    let changed = false;
    results.forEach((result, idx) => {
      if (result.status !== "fulfilled") {
        return;
      }
      const symbol = symbols[idx];
      const quote = result.value || {};
      const nextPrice = Number(quote.price || 0);
      if (!Number.isFinite(nextPrice) || nextPrice <= 0) {
        return;
      }
      const prevPrice = Number(gridQuotePriceBySymbol[symbol] || 0);
      if (nextPrice !== prevPrice) {
        changed = true;
      }
      gridQuotePriceBySymbol[symbol] = nextPrice;
    });
    clearStaleGridQuoteCache();
    if (changed) {
      renderStrategyGrid(latestStateSnapshot);
    }
  } finally {
    gridQuoteRefreshInFlight = false;
  }
}

function renderStrategyGrid(state) {
  renderConfiguredPrincipalTotal();
  if (!strategyGridBody) {
    return;
  }
  if (strategyUniverse.length === 0) {
    strategyGridBody.innerHTML = `<tr><td colspan="16" class="cell-note">등록된 종목이 없습니다. 상단에서 종목을 추가해 주세요.</td></tr>`;
    return;
  }

  const positions = Array.isArray(state.positions) ? state.positions : [];
  const recentOrders = Array.isArray(state.recent_orders) ? state.recent_orders : [];
  const posBySymbol = new Map();
  const latestOrderBySymbol = new Map();
  const buyRoundBySymbol = new Map();
  const realizedPnLBySymbol = new Map();
  const filledOrdersBySymbol = new Map();
  const sellSeenBySymbol = new Set();

  positions.forEach((p) => {
    const symbol = String(p.symbol || "").trim().toUpperCase();
    if (!symbol) {
      return;
    }
    posBySymbol.set(symbol, p);
  });
  recentOrders.forEach((o) => {
    const symbol = String(o.symbol || "").trim().toUpperCase();
    if (!symbol || latestOrderBySymbol.has(symbol)) {
      return;
    }
    latestOrderBySymbol.set(symbol, o);
  });
  recentOrders.forEach((o) => {
    const symbol = String(o.symbol || "").trim().toUpperCase();
    const status = String(o.status || "").trim().toUpperCase();
    const side = String(o.side || "").trim().toUpperCase();
    if (!symbol || status !== "FILLED") {
      return;
    }
    const qty = Number(o.qty || 0);
    const fillPrice = Number(o.fill_price || 0);
    if (!Number.isFinite(qty) || qty <= 0 || !Number.isFinite(fillPrice) || fillPrice <= 0) {
      return;
    }

    const symbolOrders = filledOrdersBySymbol.get(symbol) || [];
    symbolOrders.push(o);
    filledOrdersBySymbol.set(symbol, symbolOrders);
    if (side === "BUY") {
      if (!sellSeenBySymbol.has(symbol)) {
        const prevRound = Number(buyRoundBySymbol.get(symbol) || 0);
        buyRoundBySymbol.set(symbol, prevRound + 1);
      }
      return;
    }
    if (side === "SELL") {
      sellSeenBySymbol.add(symbol);
    }
  });
  filledOrdersBySymbol.forEach((orders, symbol) => {
    let openQty = 0;
    let openAvg = 0;
    let realizedPnL = 0;
    for (let i = orders.length - 1; i >= 0; i -= 1) {
      const order = orders[i];
      const side = String(order.side || "").trim().toUpperCase();
      const qty = Number(order.qty || 0);
      const fillPrice = Number(order.fill_price || 0);
      if (!Number.isFinite(qty) || qty <= 0 || !Number.isFinite(fillPrice) || fillPrice <= 0) {
        continue;
      }
      if (side === "BUY") {
        const totalCost = openAvg * openQty + fillPrice * qty;
        openQty += qty;
        openAvg = openQty > 0 ? totalCost / openQty : 0;
        continue;
      }
      if (side === "SELL" && openQty > 0) {
        const sellQty = Math.min(qty, openQty);
        if (sellQty <= 0) {
          continue;
        }
        realizedPnL += (fillPrice - openAvg) * sellQty;
        openQty -= sellQty;
        if (openQty <= 0) {
          openQty = 0;
          openAvg = 0;
        }
      }
    }
    realizedPnLBySymbol.set(symbol, realizedPnL);
  });

  strategyGridBody.innerHTML = "";
  strategyUniverse.forEach((symbol, idx) => {
    const position = posBySymbol.get(symbol);
    const latestOrder = latestOrderBySymbol.get(symbol);
    const cfg = getSymbolConfig(symbol);

    const qty = Number(position?.qty || 0);
    const avgPrice = Number(position?.avg_price || 0);
    const liveQuotePrice = Number(gridQuotePriceBySymbol[symbol] || 0);
    const lastPrice = liveQuotePrice > 0 ? liveQuotePrice : Number(position?.last_price || 0);
    const unrealizedPnL =
      qty > 0 && avgPrice > 0 && lastPrice > 0
        ? (lastPrice - avgPrice) * qty
        : Number(position?.unrealized_pnl || 0);
    const principalKRW = cfg.principal_krw;
    const realizedPnLKRW = Number(realizedPnLBySymbol.get(symbol) || 0);
    const appliedPrincipalKRW = Math.max(0, Math.round(principalKRW + realizedPnLKRW));
    const usedPrincipalKRW = qty > 0 && avgPrice > 0 ? qty * avgPrice : 0;
    const remainingKRW = appliedPrincipalKRW - usedPrincipalKRW;
    const remainingClass = remainingKRW >= 0 ? "profit" : "loss";
    const splitCount = cfg.split_count;
    const displayName = normalizeBoardSymbolName(cfg.display_name, symbol);
    const isSelected = normalizeSelectedValue(cfg.is_selected);
    const progressState = normalizeProgressState(cfg.progress_state);
    const progressLabel = progressState === PROGRESS_STATE_RUN ? "진행" : "대기";
    const tradeMethod = normalizeTradeMethodValue(cfg.trade_method);
    const noteText = normalizeNoteTextValue(cfg.note_text);
    const invested = qty > 0 && avgPrice > 0 ? qty * avgPrice : 0;
    const returnRate = invested > 0 ? (unrealizedPnL / invested) * 100 : NaN;
    const buyRound = Number(buyRoundBySymbol.get(symbol) || 0);
    const firstBuyCountLabel = `${buyRound}회차`;
    const recentFill = latestOrder ? `${latestOrder.side} ${formatDateTime(latestOrder.filled_at).slice(11)}` : "-";
    const defaultNote = qty > 0 ? "보유" : "초기";
    const note = noteText || defaultNote;

    const row = document.createElement("tr");
    row.innerHTML = `
      <td><input type="checkbox" class="row-symbol-check" data-symbol="${symbol}" aria-label="${symbol} 선택" ${isSelected ? "checked" : ""} /></td>
      <td>${idx + 1}</td>
      <td>${symbol}</td>
      <td>${displayName}</td>
      <td class="cell-progress">
        <div class="principal-cell">
          <input type="number" class="row-principal-input" data-symbol="${symbol}" min="1" step="1000" value="${principalKRW}" aria-label="${symbol} 종목별 원금" />
          <span class="remaining-budget ${remainingClass}">적용 ${won(appliedPrincipalKRW)}원 | 남은 ${won(remainingKRW)}원</span>
        </div>
      </td>
      <td class="cell-progress">
        <button type="button" class="row-progress-toggle ${progressState === PROGRESS_STATE_RUN ? "is-run" : "is-wait"}" data-symbol="${symbol}" data-state="${progressState}" aria-label="${symbol} 진행 상태 토글">${progressLabel}</button>
      </td>
      <td class="${invested > 0 ? "" : "cell-na"}">${invested > 0 ? won(invested) : "#N/A"}</td>
      <td class="${invested > 0 ? (unrealizedPnL >= 0 ? "profit" : "loss") : "cell-na"}">${invested > 0 ? won(unrealizedPnL) : "#N/A"}</td>
      <td class="${lastPrice > 0 ? "" : "cell-na"}">${lastPrice > 0 ? won(lastPrice) : "#N/A"}</td>
      <td class="${Number.isFinite(returnRate) ? (returnRate >= 0 ? "profit" : "loss") : "cell-na"}">${formatSignedPercent(returnRate)}</td>
      <td class="cell-buy">${firstBuyCountLabel}</td>
      <td class="cell-progress">${qty}</td>
      <td class="cell-sell">${recentFill}</td>
      <td class="cell-method">
        <input type="text" class="row-trade-method-input" data-symbol="${symbol}" maxlength="32" value="${tradeMethod}" aria-label="${symbol} 매매법" />
      </td>
      <td>
        <input type="number" class="row-split-input" data-symbol="${symbol}" min="10" max="50" step="1" value="${splitCount}" aria-label="${symbol} 입금 분할" />
      </td>
      <td class="cell-note">
        <input type="text" class="row-note-input" data-symbol="${symbol}" maxlength="255" value="${note}" aria-label="${symbol} 비고" />
      </td>
    `;
    strategyGridBody.appendChild(row);
  });
}

async function resolveBoardSymbolDisplayName(symbol) {
  const normalized = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(normalized)) {
    return normalized;
  }
  try {
    const items = await fetchSymbolMatches(normalized);
    const exact = items.find((item) => normalizeBoardSymbol(item.symbol) === normalized);
    if (exact) {
      return normalizeBoardSymbolName(exact.name, normalized);
    }
  } catch (_err) {
    // fallback below
  }
  return normalized;
}

function closeSymbolManageNameSuggestions() {
  symbolManageNameSuggestionItems = [];
  symbolManageNameSuggestionIndex = -1;
  if (!symbolManageNameSuggestions) {
    return;
  }
  symbolManageNameSuggestions.innerHTML = "";
  symbolManageNameSuggestions.hidden = true;
}

function setActiveSymbolManageSuggestion(index) {
  if (!symbolManageNameSuggestions) {
    return;
  }
  const buttons = Array.from(symbolManageNameSuggestions.querySelectorAll("button.symbol-suggestion-item"));
  buttons.forEach((button, idx) => {
    button.classList.toggle("is-active", idx === index);
  });
  symbolManageNameSuggestionIndex = index;
}

function applySymbolManageMatch(matched) {
  if (!symbolManageCodeInput || !symbolManageNameInput || !matched) {
    return false;
  }
  const symbol = normalizeBoardSymbol(matched.symbol);
  if (!isValidBoardSymbol(symbol)) {
    setSymbolManageMessage("조회된 종목코드 형식이 올바르지 않습니다.", "error");
    return false;
  }
  symbolManageCodeInput.value = symbol;
  symbolManageNameInput.value = normalizeBoardSymbolName(matched.name, symbol);
  return true;
}

function renderSymbolManageNameSuggestions(items) {
  if (!symbolManageNameSuggestions) {
    return;
  }
  symbolManageNameSuggestions.innerHTML = "";
  symbolManageNameSuggestionItems = Array.isArray(items) ? items.filter((item) => item && item.symbol) : [];
  symbolManageNameSuggestionIndex = -1;

  if (symbolManageNameSuggestionItems.length === 0) {
    const empty = document.createElement("div");
    empty.className = "symbol-suggestion-empty";
    empty.textContent = "검색 결과가 없습니다.";
    symbolManageNameSuggestions.appendChild(empty);
    symbolManageNameSuggestions.hidden = false;
    return;
  }

  symbolManageNameSuggestionItems.forEach((item, idx) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "symbol-suggestion-item";
    button.dataset.index = String(idx);

    const code = document.createElement("span");
    code.className = "symbol-suggestion-code";
    code.textContent = String(item.symbol || "");

    const name = document.createElement("span");
    name.className = "symbol-suggestion-name";
    name.textContent = String(item.name || "");

    const market = document.createElement("span");
    market.className = "symbol-suggestion-market";
    market.textContent = String(item.market || "");

    button.appendChild(code);
    button.appendChild(name);
    button.appendChild(market);
    symbolManageNameSuggestions.appendChild(button);
  });
  symbolManageNameSuggestions.hidden = false;
  setActiveSymbolManageSuggestion(0);
}

async function searchSymbolManageNameSuggestions(query) {
  const q = String(query || "").trim();
  if (q === "") {
    closeSymbolManageNameSuggestions();
    return;
  }
  try {
    const items = await fetchSymbolMatches(q);
    if (!symbolManageNameInput || String(symbolManageNameInput.value || "").trim() !== q) {
      return;
    }
    renderSymbolManageNameSuggestions(items);
  } catch (_err) {
    closeSymbolManageNameSuggestions();
  }
}

async function fillSymbolManageCodeByName() {
  if (!symbolManageCodeInput || !symbolManageNameInput) {
    return false;
  }
  const query = String(symbolManageNameInput.value || "").trim();
  if (query === "") {
    symbolManageCodeInput.value = "";
    closeSymbolManageNameSuggestions();
    return false;
  }

  let matched = null;
  if (symbolManageNameSuggestionItems.length > 0) {
    const idx = symbolManageNameSuggestionIndex >= 0 ? symbolManageNameSuggestionIndex : 0;
    if (idx >= 0 && idx < symbolManageNameSuggestionItems.length) {
      matched = symbolManageNameSuggestionItems[idx];
    }
  }

  try {
    if (!matched) {
      const items = await fetchSymbolMatches(query);
      renderSymbolManageNameSuggestions(items);
      matched = pickBestSymbolMatch(query, items);
    }
    if (!matched || !matched.symbol) {
      setSymbolManageMessage("종목명에 해당하는 종목코드를 찾지 못했습니다.", "error");
      return false;
    }
    if (!applySymbolManageMatch(matched)) {
      return false;
    }
    closeSymbolManageNameSuggestions();
    return true;
  } catch (_err) {
    setSymbolManageMessage("종목명 조회에 실패했습니다. 잠시 후 다시 시도해 주세요.", "error");
    return false;
  }
}

async function addBoardSymbol() {
  if (!symbolManageCodeInput) {
    return;
  }
  let symbol = extractBoardSymbolCandidate(symbolManageCodeInput.value);
  const inlineName = extractInlineBoardName(symbolManageCodeInput.value);
  if (!symbol && symbolManageNameInput && String(symbolManageNameInput.value || "").trim() !== "") {
    const resolved = await fillSymbolManageCodeByName();
    if (resolved) {
      symbol = extractBoardSymbolCandidate(symbolManageCodeInput.value);
    }
  }
  if (!symbol) {
    setSymbolManageMessage("추가할 종목 코드를 입력해 주세요.", "error");
    return;
  }
  if (!isValidBoardSymbol(symbol)) {
    setSymbolManageMessage("종목 코드는 영문/숫자 1~12자리만 허용됩니다.", "error");
    return;
  }
  if (strategyUniverse.includes(symbol)) {
    setSymbolManageMessage(`이미 등록된 종목입니다: ${symbol}`, "error");
    return;
  }

  const prevUniverse = [...strategyUniverse];
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  let displayName = normalizeBoardSymbolName(
    symbolManageNameInput ? symbolManageNameInput.value : inlineName,
    symbol
  );
  if (displayName === symbol) {
    displayName = await resolveBoardSymbolDisplayName(symbol);
  }
  strategyUniverse = [...strategyUniverse, symbol];
  syncSymbolConfigsWithUniverse();
  symbolConfigByCode[symbol] = {
    principal_krw: symbolConfigByCode[symbol].principal_krw,
    split_count: symbolConfigByCode[symbol].split_count,
    display_name: normalizeBoardSymbolName(displayName, symbol),
    is_selected: false,
    progress_state: DEFAULT_PROGRESS_STATE,
    sell_ratio_pct: DEFAULT_SELL_RATIO_PCT,
    trade_method: DEFAULT_TRADE_METHOD,
    note_text: "",
  };
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    strategyUniverse = prevUniverse;
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || "종목 추가 DB 저장에 실패했습니다.", "error");
    return;
  }

  symbolManageCodeInput.value = "";
  if (symbolManageNameInput) {
    symbolManageNameInput.value = "";
  }
  setSymbolManageMessage(`종목이 추가되었습니다: ${symbol} (${symbolConfigByCode[symbol].display_name})`, "ok");
  void refreshGridQuotePrices();
  startGridQuoteAutoRefresh();
}

async function runBuyRuleForRunningSymbols() {
  const runnableSymbols = strategyUniverse.filter((symbol) => {
    const cfg = getSymbolConfig(symbol);
    return normalizeProgressState(cfg.progress_state) === PROGRESS_STATE_RUN;
  });
  if (runnableSymbols.length === 0) {
    setSymbolManageMessage("진행 상태(진행)로 설정된 종목이 없습니다.", "error");
    return;
  }

  const items = runnableSymbols.map((symbol) => {
    const cfg = getSymbolConfig(symbol);
    return {
      symbol,
      display_name: normalizeBoardSymbolName(cfg.display_name, symbol),
      principal_krw: normalizePrincipalValue(cfg.principal_krw),
      split_count: normalizeSplitValue(cfg.split_count),
    };
  });

  setSymbolManageMessage(`규칙 실행 중... (${items.length}개 종목)`);
  try {
    const res = await fetch("/paper/buyrule/execute", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ items }),
    });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) {
      setSymbolManageMessage(body.message || "규칙 실행에 실패했습니다.", "error");
      return;
    }

    const totalOrders = Number(body.total_orders || 0);
    const buyOrders = Number(body.total_buy_orders || 0);
    const sellOrders = Number(body.total_sell_orders || 0);
    setSymbolManageMessage(
      `규칙 실행 완료: 주문 ${totalOrders}건 (매수 ${buyOrders}, 매도 ${sellOrders})`,
      totalOrders > 0 ? "ok" : ""
    );
    await refreshState();
  } catch (_err) {
    setSymbolManageMessage("규칙 실행 중 네트워크 오류가 발생했습니다.", "error");
  }
}

async function removeSelectedBoardSymbols() {
  if (!strategyGridBody) {
    return;
  }
  const checked = Array.from(strategyGridBody.querySelectorAll("input.row-symbol-check:checked"));
  if (checked.length === 0) {
    setSymbolManageMessage("삭제할 종목을 선택해 주세요.", "error");
    return;
  }
  const targetSymbols = new Set(
    checked
      .map((node) => normalizeBoardSymbol(node.getAttribute("data-symbol")))
      .filter((symbol) => symbol !== "")
  );
  const prevUniverse = [...strategyUniverse];
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  strategyUniverse = strategyUniverse.filter((symbol) => !targetSymbols.has(symbol));
  syncSymbolConfigsWithUniverse();
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    strategyUniverse = prevUniverse;
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || "종목 삭제 DB 저장에 실패했습니다.", "error");
    return;
  }
  setSymbolManageMessage(`${targetSymbols.size}개 종목을 삭제했습니다.`, "ok");
  void refreshGridQuotePrices();
  startGridQuoteAutoRefresh();
}

async function resetBoardSymbols() {
  const prevUniverse = [...strategyUniverse];
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  strategyUniverse = [...DEFAULT_STRATEGY_UNIVERSE];
  syncSymbolConfigsWithUniverse();
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    strategyUniverse = prevUniverse;
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || "기본 종목 복원 DB 저장에 실패했습니다.", "error");
    return;
  }
  setSymbolManageMessage("기본 종목 목록으로 복원했습니다.", "ok");
  void refreshGridQuotePrices();
  startGridQuoteAutoRefresh();
}

async function updateBoardSymbolPrincipal(symbol, rawValue) {
  const key = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(key)) {
    return;
  }
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  const principalKRW = normalizePrincipalValue(rawValue);
  const prev = getSymbolConfig(key);
  symbolConfigByCode[key] = { ...prev, principal_krw: principalKRW };
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || `${key} 원금 DB 저장에 실패했습니다.`, "error");
    return;
  }
  setSymbolManageMessage(`${key} 원금을 ${won(principalKRW)}원으로 저장했습니다.`, "ok");
}

async function updateBoardSymbolSplit(symbol, rawValue) {
  const key = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(key)) {
    return;
  }
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  const rawNumber = Number(rawValue);
  const splitCount = normalizeSplitValue(rawValue);
  const prev = getSymbolConfig(key);
  symbolConfigByCode[key] = { ...prev, split_count: splitCount };
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || `${key} 입금 분할 DB 저장에 실패했습니다.`, "error");
    return;
  }
  if (!Number.isFinite(rawNumber) || rawNumber < MIN_SPLIT_COUNT || rawNumber > MAX_SPLIT_COUNT) {
    setSymbolManageMessage(`${key} 입금 분할은 10~50만 허용됩니다. ${splitCount}로 조정했습니다.`, "error");
    return;
  }
  setSymbolManageMessage(`${key} 입금 분할을 ${splitCount}으로 저장했습니다.`, "ok");
}

async function updateBoardSymbolChecked(symbol, checked) {
  const key = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(key)) {
    return;
  }
  const nextChecked = normalizeSelectedValue(checked);
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  const prev = getSymbolConfig(key);
  if (normalizeSelectedValue(prev.is_selected) === nextChecked) {
    return;
  }
  symbolConfigByCode[key] = { ...prev, is_selected: nextChecked };
  try {
    await persistBoardState();
  } catch (error) {
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || `${key} 체크 상태 DB 저장에 실패했습니다.`, "error");
  }
}

async function toggleBoardSymbolProgress(symbol) {
  const key = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(key)) {
    return;
  }
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  const prev = getSymbolConfig(key);
  const nextProgress =
    normalizeProgressState(prev.progress_state) === PROGRESS_STATE_RUN
      ? PROGRESS_STATE_WAIT
      : PROGRESS_STATE_RUN;
  symbolConfigByCode[key] = { ...prev, progress_state: nextProgress };
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || `${key} 진행 상태 DB 저장에 실패했습니다.`, "error");
    return;
  }
  setSymbolManageMessage(`${key} 진행 상태를 ${nextProgress === PROGRESS_STATE_RUN ? "진행" : "대기"}으로 저장했습니다.`, "ok");
}

async function updateBoardSymbolSellRatio(symbol, rawValue) {
  const key = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(key)) {
    return;
  }
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  const rawNumber = Number(rawValue);
  const sellRatioPct = normalizeSellRatioValue(rawValue);
  const prev = getSymbolConfig(key);
  symbolConfigByCode[key] = { ...prev, sell_ratio_pct: sellRatioPct };
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || `${key} 매도가 비율 DB 저장에 실패했습니다.`, "error");
    return;
  }
  if (!Number.isFinite(rawNumber) || rawNumber < MIN_SELL_RATIO_PCT || rawNumber > MAX_SELL_RATIO_PCT) {
    setSymbolManageMessage(
      `${key} 매도가 비율은 ${MIN_SELL_RATIO_PCT}~${MAX_SELL_RATIO_PCT}만 허용됩니다. ${sellRatioPct}%로 조정했습니다.`,
      "error"
    );
    return;
  }
  setSymbolManageMessage(`${key} 매도가 비율을 ${sellRatioPct}%로 저장했습니다.`, "ok");
}

async function updateBoardSymbolTradeMethod(symbol, rawValue) {
  const key = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(key)) {
    return;
  }
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  const tradeMethod = normalizeTradeMethodValue(rawValue);
  const prev = getSymbolConfig(key);
  symbolConfigByCode[key] = { ...prev, trade_method: tradeMethod };
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || `${key} 매매법 DB 저장에 실패했습니다.`, "error");
    return;
  }
  setSymbolManageMessage(`${key} 매매법을 ${tradeMethod}로 저장했습니다.`, "ok");
}

async function updateBoardSymbolNote(symbol, rawValue) {
  const key = normalizeBoardSymbol(symbol);
  if (!isValidBoardSymbol(key)) {
    return;
  }
  const prevConfig = cloneSymbolConfigMap(symbolConfigByCode);
  const noteText = normalizeNoteTextValue(rawValue);
  const prev = getSymbolConfig(key);
  symbolConfigByCode[key] = { ...prev, note_text: noteText };
  renderStrategyGrid(latestStateSnapshot);
  try {
    await persistBoardState();
  } catch (error) {
    symbolConfigByCode = prevConfig;
    renderStrategyGrid(latestStateSnapshot);
    setSymbolManageMessage(error.message || `${key} 비고 DB 저장에 실패했습니다.`, "error");
    return;
  }
  setSymbolManageMessage(`${key} 비고를 저장했습니다.`, "ok");
}

async function fetchQuote(symbol) {
  const res = await fetch(`/market/quote?symbol=${encodeURIComponent(symbol)}`);
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.message || "시세 조회 실패");
  }
  return body;
}

function rangeToLimit(range) {
  switch (range) {
    case "1d":
      return 500;
    case "1w":
      return 35;
    case "1m":
      return 30;
    case "3m":
      return 90;
    case "1y":
      return 365;
    default:
      return 30;
  }
}

function rangeToInterval(range) {
  switch (range) {
    case "1d":
      return "1m";
    case "1w":
      return "1h";
    default:
      return "1d";
  }
}

async function fetchCandles(symbol, range) {
  const limit = rangeToLimit(range);
  const interval = rangeToInterval(range);
  const res = await fetch(`/market/candles?symbol=${encodeURIComponent(symbol)}&interval=${encodeURIComponent(interval)}&limit=${limit}`);
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.message || "캔들 조회 실패");
  }
  return body;
}

async function fetchSymbolMatches(query) {
  const res = await fetch(`/market/symbols/search?query=${encodeURIComponent(query)}&limit=8`);
  const body = await res.json();
  if (!res.ok) {
    return [];
  }
  if (!body || !Array.isArray(body.items)) {
    return [];
  }
  return body.items;
}

function normalizeLookupText(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "");
}

function normalizeTickerCode(value) {
  return String(value || "")
    .replace(/\D/g, "")
    .slice(0, 6);
}

function renderQuoteNameSuggestions(items) {
  if (!quoteSymbolNameOptions) {
    return;
  }
  quoteSymbolNameOptions.innerHTML = "";
  items.forEach((item) => {
    const option = document.createElement("option");
    option.value = item.name;
    option.label = `${item.name} (${item.symbol})`;
    quoteSymbolNameOptions.appendChild(option);
  });
}

function pickBestSymbolMatch(query, items) {
  if (!Array.isArray(items) || items.length === 0) {
    return null;
  }
  const raw = String(query || "").trim();
  if (!raw) {
    return items[0];
  }
  const code = normalizeTickerCode(raw);
  if (code.length === 6) {
    const exactByCode = items.find((item) => String(item.symbol || "").trim() === code);
    if (exactByCode) {
      return exactByCode;
    }
  }
  const nq = normalizeLookupText(raw);
  const exactByName = items.find((item) => normalizeLookupText(item.name) === nq);
  if (exactByName) {
    return exactByName;
  }
  return items[0];
}

async function resolveTickerFromName(name) {
  const query = String(name || "").trim();
  if (!query) {
    return null;
  }
  const items = await fetchSymbolMatches(query);
  renderQuoteNameSuggestions(items);
  return pickBestSymbolMatch(query, items);
}

async function resolveNameFromTicker(code) {
  const ticker = normalizeTickerCode(code);
  if (!ticker || ticker.length !== 6) {
    return null;
  }
  const items = await fetchSymbolMatches(ticker);
  renderQuoteNameSuggestions(items);
  const exact = items.find((item) => String(item.symbol || "").trim() === ticker);
  return exact || pickBestSymbolMatch(ticker, items);
}

function normalizeCandles(candles) {
  if (!Array.isArray(candles)) {
    return [];
  }
  const normalized = candles
    .map((c) => ({
      ...c,
      _ts: new Date(c.time).getTime(),
    }))
    .filter((c) => Number.isFinite(c._ts) && Number.isFinite(Number(c.close || 0)) && Number(c.close || 0) > 0)
    .sort((a, b) => a._ts - b._ts);
  return normalized.map((c) => {
    const copy = { ...c };
    delete copy._ts;
    return copy;
  });
}

function candlesForRange(candles, range) {
  const sorted = normalizeCandles(candles);
  if (sorted.length === 0) {
    return sorted;
  }
  if (range === "1d") {
    const latestParts = toKSTParts(sorted[sorted.length - 1].time);
    if (latestParts) {
      const sameDay = sorted.filter((c) => {
        const p = toKSTParts(c.time);
        return p && p.date === latestParts.date;
      });
      const sessionCandles = sameDay.filter((c) => isKrxRegularSession(toKSTParts(c.time)));
      if (sessionCandles.length >= 2) {
        return sessionCandles;
      }
      if (sameDay.length >= 2) {
        return sameDay;
      }
    }
  }
  return sorted;
}

function midPrice(candle) {
  const high = Number(candle.high || 0);
  const low = Number(candle.low || 0);
  if (high > 0 && low > 0) {
    return (high + low) / 2;
  }
  return Number(candle.close || 0);
}

function toTrendPoints(candles, range) {
  if (!Array.isArray(candles) || candles.length === 0) {
    return [];
  }

  return candles
    .map((c) => ({
      time: c.time,
      price: midPrice(c),
    }))
    .sort((a, b) => new Date(a.time).getTime() - new Date(b.time).getTime())
    .filter((p) => Number.isFinite(p.price) && p.price > 0);
}

function computeRangeChange(candles) {
  if (!Array.isArray(candles) || candles.length < 2) {
    return null;
  }
  const first = Number(candles[0].close || 0);
  const last = Number(candles[candles.length - 1].close || 0);
  if (first <= 0 || last <= 0) {
    return null;
  }
  const diff = last - first;
  const pct = (diff / first) * 100;
  return { first, last, diff, pct };
}

function renderQuote(quote, candles, range) {
  quoteResult.className = "muted";
  const asOfText = formatDateTime(quote.as_of);
  const chg = computeRangeChange(candles);
  if (!chg) {
    quoteResult.textContent = `종목 ${quote.symbol} | 현재가 ${won(quote.price)} | 기준시각 ${asOfText}`;
    return;
  }
  const sign = chg.diff >= 0 ? "+" : "";
  quoteResult.textContent = `종목 ${quote.symbol} | 현재가 ${won(quote.price)} | ${range} 변화 ${sign}${won(chg.diff)} (${sign}${chg.pct.toFixed(2)}%) | 기준시각 ${asOfText}`;
}

function formatChartLabel(timeValue, range) {
  const d = new Date(timeValue);
  if (!Number.isFinite(d.getTime())) {
    return "";
  }
  if (range === "1d") {
    return d.toLocaleTimeString("ko-KR", {
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
      timeZone: "Asia/Seoul",
    });
  }
  return d.toLocaleDateString("ko-KR", {
    month: "2-digit",
    day: "2-digit",
    timeZone: "Asia/Seoul",
  });
}

function toKSTParts(timeValue) {
  const d = new Date(timeValue);
  if (!Number.isFinite(d.getTime())) {
    return null;
  }
  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Seoul",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  });
  const parts = formatter.formatToParts(d);
  const map = {};
  parts.forEach((p) => {
    if (p.type !== "literal") {
      map[p.type] = p.value;
    }
  });
  if (!map.year || !map.month || !map.day || !map.hour || !map.minute) {
    return null;
  }
  return {
    date: `${map.year}-${map.month}-${map.day}`,
    hour: Number(map.hour),
    minute: Number(map.minute),
  };
}

function isKrxRegularSession(parts) {
  if (!parts) {
    return false;
  }
  const hhmm = parts.hour * 100 + parts.minute;
  return hhmm >= 900 && hhmm <= 1530;
}

function renderCandles(candles, range = "1m", prebuiltPoints = null) {
  if (!candleChart) {
    return;
  }
  candleChart.innerHTML = "";
  const points = Array.isArray(prebuiltPoints) ? prebuiltPoints : toTrendPoints(candles, range);
  if (points.length === 0) {
    candleChart.innerHTML = `<text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="#5b6661">차트 데이터가 없습니다.</text>`;
    return;
  }

  const chartWidth = 900;
  const chartHeight = 280;
  const padLeft = 48;
  const padRight = 16;
  const padTop = 10;
  const padBottom = 24;
  const drawWidth = chartWidth - padLeft - padRight;
  const drawHeight = chartHeight - padTop - padBottom;
  const prices = points.map((p) => Number(p.price || 0));
  let maxPrice = Math.max(...prices);
  let minPrice = Math.min(...prices);
  if (maxPrice <= minPrice) {
    maxPrice = minPrice + 1;
  }

  const toY = (price) => {
    const ratio = (price - minPrice) / (maxPrice - minPrice);
    return padTop + (1 - ratio) * drawHeight;
  };

  const step = drawWidth / Math.max(1, points.length - 1);

  for (let i = 1; i < points.length; i += 1) {
    const prev = points[i - 1];
    const cur = points[i];
    const x1 = padLeft + (i - 1) * step;
    const y1 = toY(prev.price);
    const x2 = padLeft + i * step;
    const y2 = toY(cur.price);
    const seg = document.createElementNS("http://www.w3.org/2000/svg", "line");
    seg.setAttribute("x1", x1);
    seg.setAttribute("y1", y1);
    seg.setAttribute("x2", x2);
    seg.setAttribute("y2", y2);
    seg.setAttribute("stroke", cur.price >= prev.price ? "#c73b3b" : "#1f6c5b");
    seg.setAttribute("stroke-width", "2");
    candleChart.appendChild(seg);
  }
  if (points.length <= 120) {
    points.forEach((p, i) => {
      const dot = document.createElementNS("http://www.w3.org/2000/svg", "circle");
      dot.setAttribute("cx", padLeft + i * step);
      dot.setAttribute("cy", toY(p.price));
      dot.setAttribute("r", "1.6");
      dot.setAttribute("fill", "#1f2522");
      dot.setAttribute("opacity", "0.45");
      candleChart.appendChild(dot);
    });
  }

  const axis = document.createElementNS("http://www.w3.org/2000/svg", "line");
  axis.setAttribute("x1", padLeft);
  axis.setAttribute("x2", chartWidth - padRight);
  axis.setAttribute("y1", chartHeight - padBottom);
  axis.setAttribute("y2", chartHeight - padBottom);
  axis.setAttribute("stroke", "#dde3de");
  axis.setAttribute("stroke-width", "1");
  candleChart.appendChild(axis);

  const tickIndexes = [];
  if (range === "1d") {
    // 1일(09:00~15:30 1분선): 장중 30분 간격 눈금 표시
    points.forEach((p, idx) => {
      const parts = toKSTParts(p.time);
      if (!parts) {
        return;
      }
      if (!isKrxRegularSession(parts)) {
        return;
      }
      if (parts.minute === 0 || parts.minute === 30) {
        tickIndexes.push(idx);
      }
    });
    if (tickIndexes.length < 2) {
      const stride = Math.max(1, Math.floor(points.length / 6));
      for (let i = 0; i < points.length; i += stride) {
        tickIndexes.push(i);
      }
    }
  } else {
    const stride = Math.max(1, Math.floor(points.length / 6));
    for (let i = 0; i < points.length; i += stride) {
      tickIndexes.push(i);
    }
  }
  if (tickIndexes[tickIndexes.length - 1] !== points.length - 1) {
    tickIndexes.push(points.length - 1);
  }

  tickIndexes.forEach((idx) => {
    if (idx < 0 || idx >= points.length) {
      return;
    }
    const x = padLeft + idx * step;
    const tick = document.createElementNS("http://www.w3.org/2000/svg", "line");
    tick.setAttribute("x1", x);
    tick.setAttribute("x2", x);
    tick.setAttribute("y1", chartHeight - padBottom);
    tick.setAttribute("y2", chartHeight - padBottom + 4);
    tick.setAttribute("stroke", "#dde3de");
    tick.setAttribute("stroke-width", "1");
    candleChart.appendChild(tick);

    const label = document.createElementNS("http://www.w3.org/2000/svg", "text");
    label.setAttribute("x", x);
    label.setAttribute("y", chartHeight - 6);
    label.setAttribute("fill", "#5b6661");
    label.setAttribute("font-size", "10");
    label.setAttribute("text-anchor", "middle");
    label.textContent = formatChartLabel(points[idx].time, range);
    candleChart.appendChild(label);
  });

  const topLabel = document.createElementNS("http://www.w3.org/2000/svg", "text");
  topLabel.setAttribute("x", 4);
  topLabel.setAttribute("y", padTop + 10);
  topLabel.setAttribute("fill", "#5b6661");
  topLabel.setAttribute("font-size", "11");
  topLabel.textContent = won(maxPrice);
  candleChart.appendChild(topLabel);

  const bottomLabel = document.createElementNS("http://www.w3.org/2000/svg", "text");
  bottomLabel.setAttribute("x", 4);
  bottomLabel.setAttribute("y", chartHeight - padBottom);
  bottomLabel.setAttribute("fill", "#5b6661");
  bottomLabel.setAttribute("font-size", "11");
  bottomLabel.textContent = won(minPrice);
  candleChart.appendChild(bottomLabel);
}

async function refreshQuotePanel(symbol, range) {
  quoteMessage.className = "muted";
  quoteMessage.textContent = "시세/캔들 조회 중...";
  const [quote, candles] = await Promise.all([
    fetchQuote(symbol),
    fetchCandles(symbol, range),
  ]);
  const rawCount = Array.isArray(candles) ? candles.length : 0;
  const scopedCandles = candlesForRange(candles, range);
  const trendPoints = toTrendPoints(scopedCandles, range);
  renderQuote(quote, scopedCandles, range);
  renderCandles(scopedCandles, range, trendPoints);
  quoteMessage.className = "profit";
  quoteMessage.textContent = `조회 성공 (${rangeToInterval(range)}, ${range}) - 수신 ${rawCount}개, 범위 ${scopedCandles.length}개, 표시 ${trendPoints.length}개`;
}

function renderState(state) {
  latestStateSnapshot = {
    cash: Number(state.cash || 0),
    realized_pnl_today: Number(state.realized_pnl_today || 0),
    positions: Array.isArray(state.positions) ? [...state.positions] : [],
    recent_orders: Array.isArray(state.recent_orders) ? [...state.recent_orders] : [],
  };

  const positions = Array.isArray(state.positions) ? state.positions : [];
  const recentOrders = Array.isArray(state.recent_orders) ? state.recent_orders : [];
  const cashValue = Number(state.cash || 0);
  const realizedPnLValue = Number(state.realized_pnl_today || 0);
  const investedValue = positions.reduce((sum, p) => {
    const qty = Number(p.qty || 0);
    const symbol = String(p.symbol || "").trim().toUpperCase();
    const liveQuotePrice = Number(gridQuotePriceBySymbol[symbol] || 0);
    const markPrice = liveQuotePrice > 0 ? liveQuotePrice : Number(p.last_price || p.avg_price || 0);
    return sum + qty * markPrice;
  }, 0);
  const totalValue = cashValue + investedValue;

  if (accountCash) {
    accountCash.textContent = won(cashValue);
  }
  renderConfiguredPrincipalTotal();
  if (accountTotal) {
    accountTotal.textContent = won(totalValue);
  }
  if (accountRealizedPnL) {
    accountRealizedPnL.textContent = won(realizedPnLValue);
    accountRealizedPnL.classList.remove("profit", "loss");
    accountRealizedPnL.classList.add(realizedPnLValue >= 0 ? "profit" : "loss");
  }
  if (accountPositionCount) {
    accountPositionCount.textContent = String(positions.length);
  }
  if (accountOrderCount) {
    accountOrderCount.textContent = String(Math.min(recentOrders.length, 10));
  }
  if (accountUpdatedAt) {
    accountUpdatedAt.textContent = `최근 갱신: ${formatKSTNow()}`;
  }
  renderStrategyGrid(state);

  if (cashSummary) {
    cashSummary.textContent = `현금: ${won(state.cash)} | 당일 실현손익: ${won(state.realized_pnl_today)}`;
  }

  if (positionsBody) {
    positionsBody.innerHTML = "";
    if (positions.length === 0) {
      positionsBody.innerHTML = `<tr><td colspan="4" class="muted">보유 포지션이 없습니다.</td></tr>`;
    } else {
      positions.forEach((p) => {
        const symbol = String(p.symbol || "").trim().toUpperCase();
        const qty = Number(p.qty || 0);
        const avgPrice = Number(p.avg_price || 0);
        const liveQuotePrice = Number(gridQuotePriceBySymbol[symbol] || 0);
        const markPrice = liveQuotePrice > 0 ? liveQuotePrice : Number(p.last_price || 0);
        const unrealizedPnL =
          qty > 0 && avgPrice > 0 && markPrice > 0
            ? (markPrice - avgPrice) * qty
            : Number(p.unrealized_pnl || 0);
        const pnlClass = unrealizedPnL >= 0 ? "profit" : "loss";
        const row = document.createElement("tr");
        row.innerHTML = `
          <td>${symbol}</td>
          <td>${qty}</td>
          <td>${won(avgPrice)}</td>
          <td class="${pnlClass}">${won(unrealizedPnL)}</td>
        `;
        positionsBody.appendChild(row);
      });
    }
  }

  if (ordersBody) {
    ordersBody.innerHTML = "";
    if (recentOrders.length === 0) {
      ordersBody.innerHTML = `<tr><td colspan="6" class="muted">주문 내역이 없습니다.</td></tr>`;
    } else {
      recentOrders.slice(0, 10).forEach((o) => {
        const filledAt = formatDateTime(o.filled_at);
        const row = document.createElement("tr");
        row.innerHTML = `
          <td>${filledAt}</td>
          <td>${o.symbol}</td>
          <td>${o.side}</td>
          <td>${o.qty}</td>
          <td>${won(o.fill_price)}</td>
          <td>${o.status}</td>
        `;
        ordersBody.appendChild(row);
      });
    }
  }
}

async function refreshState() {
  const res = await fetch("/paper/state");
  if (!res.ok) {
    throw new Error("failed to fetch mock account state");
  }
  const state = await res.json();
  renderState(state);
}

function connectSSE() {
  const source = new EventSource("/events/stream");
  source.onopen = () => setSSEStatus("SSE 연결됨", "ok");
  source.onerror = () => setSSEStatus("SSE 재연결 중", "error");

  source.addEventListener("auto_trade", (event) => {
    const data = JSON.parse(event.data);
    setLastEvent(`[auto_trade] ${data.data.signal} / ${data.data.reason}`);
  });

  source.addEventListener("paper_order", (event) => {
    const data = JSON.parse(event.data);
    setLastEvent(`[모의계좌 주문] ${data.data.side} ${data.data.symbol} x ${data.data.qty}`);
  });

  source.addEventListener("paper_state", (event) => {
    const data = JSON.parse(event.data);
    renderState(data.data);
    setLastEvent("[모의계좌 상태] 갱신");
  });

  source.addEventListener("v22_sim_tick", (event) => {
    const data = JSON.parse(event.data);
    const tick = data.data || {};
    const day = Number(tick.day || 0);
    const price = Number(tick.price || 0);
    setLastEvent(`[V2.2 시뮬] ${day}일차 / ${won(price)}원`);
    void refreshGridQuotePrices();
  });

  source.addEventListener("v22_sim_state", (event) => {
    const data = JSON.parse(event.data);
    renderV22SimState(data.data || {});
  });
}

orderForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const formData = new FormData(orderForm);
  const payload = {
    symbol: String(formData.get("symbol") || "").trim(),
    side: String(formData.get("side") || "").trim().toUpperCase(),
    qty: Number(formData.get("qty") || 0),
  };

  orderMessage.className = "muted";
  orderMessage.textContent = "주문 전송 중...";

  try {
    const res = await fetch("/paper/order", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const body = await res.json();
    if (!res.ok) {
      orderMessage.className = "loss";
      orderMessage.textContent = body.message || "주문 실패";
      return;
    }
    orderMessage.className = "profit";
    orderMessage.textContent = `주문 성공: #${body.id} ${body.side} ${body.symbol} x ${body.qty}`;
  } catch (error) {
    orderMessage.className = "loss";
    orderMessage.textContent = "주문 중 네트워크 오류가 발생했습니다.";
  }
});

quoteForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const codeValue = normalizeTickerCode(quoteSymbolCodeInput ? quoteSymbolCodeInput.value : "");
  if (quoteSymbolCodeInput) {
    quoteSymbolCodeInput.value = codeValue;
  }

  let symbol = codeValue;
  if (!symbol && quoteSymbolNameInput) {
    try {
      const matched = await resolveTickerFromName(quoteSymbolNameInput.value);
      if (matched && quoteSymbolCodeInput) {
        quoteSymbolCodeInput.value = matched.symbol;
        symbol = matched.symbol;
        if (matched.name) {
          quoteSymbolNameInput.value = matched.name;
        }
      }
    } catch (_error) {
      quoteMessage.className = "loss";
      quoteMessage.textContent = "종목명으로 티커 조회에 실패했습니다. 잠시 후 다시 시도해 주세요.";
      return;
    }
  }

  if (!symbol) {
    quoteMessage.className = "loss";
    quoteMessage.textContent = "티커 번호(6자리) 또는 종목명을 입력해 주세요.";
    return;
  }

  quoteMessage.className = "muted";
  quoteMessage.textContent = "조회 중...";
  try {
    await refreshQuotePanel(symbol, selectedRange);
  } catch (error) {
    quoteMessage.className = "loss";
    quoteMessage.textContent = error.message || "시세 조회 실패";
  }
});

if (quoteSymbolNameInput) {
  quoteSymbolNameInput.addEventListener("input", () => {
    const nameQuery = String(quoteSymbolNameInput.value || "").trim();
    if (quoteNameSearchDebounceTimer) {
      clearTimeout(quoteNameSearchDebounceTimer);
      quoteNameSearchDebounceTimer = null;
    }
    if (!nameQuery) {
      if (quoteSymbolNameOptions) {
        quoteSymbolNameOptions.innerHTML = "";
      }
      return;
    }
    quoteNameSearchDebounceTimer = setTimeout(async () => {
      try {
        const matched = await resolveTickerFromName(nameQuery);
        if (matched && quoteSymbolCodeInput) {
          quoteSymbolCodeInput.value = matched.symbol;
        }
      } catch (_err) {
        if (quoteSymbolNameOptions) {
          quoteSymbolNameOptions.innerHTML = "";
        }
      }
    }, 220);
  });

  quoteSymbolNameInput.addEventListener("change", async () => {
    const matched = await resolveTickerFromName(quoteSymbolNameInput.value);
    if (matched && quoteSymbolCodeInput) {
      quoteSymbolCodeInput.value = matched.symbol;
      if (matched.name) {
        quoteSymbolNameInput.value = matched.name;
      }
    }
  });
}

if (quoteSymbolCodeInput) {
  quoteSymbolCodeInput.addEventListener("input", () => {
    quoteSymbolCodeInput.value = normalizeTickerCode(quoteSymbolCodeInput.value);
  });

  quoteSymbolCodeInput.addEventListener("keydown", async (event) => {
    if (event.key !== "Enter") {
      return;
    }
    event.preventDefault();
    const ticker = normalizeTickerCode(quoteSymbolCodeInput.value);
    quoteSymbolCodeInput.value = ticker;
    if (ticker.length !== 6) {
      return;
    }
    try {
      const matched = await resolveNameFromTicker(ticker);
      if (matched && quoteSymbolNameInput) {
        quoteSymbolNameInput.value = matched.name || quoteSymbolNameInput.value;
      }
    } catch (_err) {
      // ignore resolve errors on enter
    }
  });
}

if (symbolManageForm) {
  symbolManageForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    await addBoardSymbol();
  });
}

if (symbolManageCodeInput) {
  symbolManageCodeInput.addEventListener("input", () => {
    symbolManageCodeInput.value = normalizeBoardSymbol(symbolManageCodeInput.value);
  });
}

if (symbolManageNameInput) {
  symbolManageNameInput.addEventListener("input", () => {
    const query = String(symbolManageNameInput.value || "").trim();
    if (symbolManageNameSearchDebounceTimer) {
      clearTimeout(symbolManageNameSearchDebounceTimer);
      symbolManageNameSearchDebounceTimer = null;
    }
    if (query === "") {
      closeSymbolManageNameSuggestions();
      return;
    }
    symbolManageNameSearchDebounceTimer = setTimeout(() => {
      searchSymbolManageNameSuggestions(query);
    }, 180);
  });

  symbolManageNameInput.addEventListener("keydown", async (event) => {
    if (event.key === "ArrowDown") {
      if (symbolManageNameSuggestionItems.length > 0) {
        event.preventDefault();
        const current = symbolManageNameSuggestionIndex >= 0 ? symbolManageNameSuggestionIndex : -1;
        const next = Math.min(symbolManageNameSuggestionItems.length - 1, current + 1);
        setActiveSymbolManageSuggestion(next);
      }
      return;
    }
    if (event.key === "ArrowUp") {
      if (symbolManageNameSuggestionItems.length > 0) {
        event.preventDefault();
        const current = symbolManageNameSuggestionIndex >= 0 ? symbolManageNameSuggestionIndex : symbolManageNameSuggestionItems.length;
        const next = Math.max(0, current - 1);
        setActiveSymbolManageSuggestion(next);
      }
      return;
    }
    if (event.key === "Escape") {
      closeSymbolManageNameSuggestions();
      return;
    }
    if (event.key !== "Enter") {
      return;
    }
    event.preventDefault();

    if (symbolManageNameSuggestionItems.length > 0) {
      const idx = symbolManageNameSuggestionIndex >= 0 ? symbolManageNameSuggestionIndex : 0;
      const matched = symbolManageNameSuggestionItems[idx];
      if (applySymbolManageMatch(matched)) {
        closeSymbolManageNameSuggestions();
        setSymbolManageMessage(`종목코드를 찾았습니다: ${symbolManageCodeInput ? symbolManageCodeInput.value : ""}`, "ok");
        return;
      }
    }

    const resolved = await fillSymbolManageCodeByName();
    if (resolved && symbolManageCodeInput) {
      setSymbolManageMessage(`종목코드를 찾았습니다: ${symbolManageCodeInput.value}`, "ok");
    }
  });

  symbolManageNameInput.addEventListener("focus", () => {
    const query = String(symbolManageNameInput.value || "").trim();
    if (query !== "" && symbolManageNameSuggestionItems.length === 0) {
      void searchSymbolManageNameSuggestions(query);
    }
  });
}

if (symbolManageNameSuggestions) {
  symbolManageNameSuggestions.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target.closest("button.symbol-suggestion-item") : null;
    if (!target) {
      return;
    }
    const idx = Number(target.getAttribute("data-index"));
    if (!Number.isInteger(idx) || idx < 0 || idx >= symbolManageNameSuggestionItems.length) {
      return;
    }
    const matched = symbolManageNameSuggestionItems[idx];
    if (applySymbolManageMatch(matched) && symbolManageCodeInput) {
      closeSymbolManageNameSuggestions();
      setSymbolManageMessage(`종목코드를 찾았습니다: ${symbolManageCodeInput.value}`, "ok");
    }
  });
}

if (symbolManageNameSearchButton) {
  symbolManageNameSearchButton.addEventListener("click", async () => {
    const resolved = await fillSymbolManageCodeByName();
    if (resolved && symbolManageCodeInput) {
      setSymbolManageMessage(`종목코드를 찾았습니다: ${symbolManageCodeInput.value}`, "ok");
    }
  });
}

document.addEventListener("click", (event) => {
  if (!symbolManageNameSuggestions || symbolManageNameSuggestions.hidden) {
    return;
  }
  const target = event.target;
  if (!(target instanceof Node)) {
    return;
  }
  if (symbolManageNameSuggestions.contains(target)) {
    return;
  }
  if (symbolManageNameInput && symbolManageNameInput.contains(target)) {
    return;
  }
  if (symbolManageNameSearchButton && symbolManageNameSearchButton.contains(target)) {
    return;
  }
  closeSymbolManageNameSuggestions();
});

if (symbolRemoveSelectedButton) {
  symbolRemoveSelectedButton.addEventListener("click", async () => {
    await removeSelectedBoardSymbols();
  });
}

if (symbolRunSelectedRuleButton) {
  symbolRunSelectedRuleButton.addEventListener("click", async () => {
    await runBuyRuleForRunningSymbols();
  });
}

if (symbolResetButton) {
  symbolResetButton.addEventListener("click", async () => {
    await resetBoardSymbols();
  });
}

if (v22SimStartButton) {
  v22SimStartButton.addEventListener("click", async () => {
    try {
      await startV22Simulation();
    } catch (error) {
      setV22SimMessage(error.message || "시뮬레이션 시작에 실패했습니다.", "error");
    }
  });
}

if (v22SimStopButton) {
  v22SimStopButton.addEventListener("click", async () => {
    try {
      await stopV22Simulation();
    } catch (error) {
      setV22SimMessage(error.message || "시뮬레이션 중지에 실패했습니다.", "error");
    }
  });
}

if (strategyGridBody) {
  strategyGridBody.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target.closest("button.row-progress-toggle") : null;
    if (!target) {
      return;
    }
    void toggleBoardSymbolProgress(target.getAttribute("data-symbol"));
  });

  strategyGridBody.addEventListener("change", (event) => {
    const target = event.target;
    if (!target || !(target instanceof HTMLInputElement)) {
      return;
    }
    if (target.classList.contains("row-symbol-check")) {
      void updateBoardSymbolChecked(target.getAttribute("data-symbol"), target.checked);
      return;
    }
    if (target.classList.contains("row-principal-input")) {
      void updateBoardSymbolPrincipal(target.getAttribute("data-symbol"), target.value);
      return;
    }
    if (target.classList.contains("row-split-input")) {
      void updateBoardSymbolSplit(target.getAttribute("data-symbol"), target.value);
      return;
    }
    if (target.classList.contains("row-sell-ratio-input")) {
      void updateBoardSymbolSellRatio(target.getAttribute("data-symbol"), target.value);
      return;
    }
    if (target.classList.contains("row-trade-method-input")) {
      void updateBoardSymbolTradeMethod(target.getAttribute("data-symbol"), target.value);
      return;
    }
    if (target.classList.contains("row-note-input")) {
      void updateBoardSymbolNote(target.getAttribute("data-symbol"), target.value);
    }
  });
}

rangeButtons.addEventListener("click", async (event) => {
  const button = event.target.closest("button[data-range]");
  if (!button) {
    return;
  }
  selectedRange = String(button.dataset.range || "1d");
  rangeButtons.querySelectorAll("button[data-range]").forEach((b) => {
    b.classList.toggle("active", b === button);
  });

  let symbol = normalizeTickerCode(quoteSymbolCodeInput ? quoteSymbolCodeInput.value : "");
  if (!symbol) {
    symbol = "005930";
    if (quoteSymbolCodeInput) {
      quoteSymbolCodeInput.value = symbol;
    }
  }
  try {
    await refreshQuotePanel(symbol, selectedRange);
  } catch (error) {
    quoteMessage.className = "loss";
    quoteMessage.textContent = error.message || "시세 조회 실패";
  }
});

async function bootstrapTradePage() {
  await bootstrapBoardState();
  renderStrategyGrid(latestStateSnapshot);
  try {
    const simState = await fetchV22SimulationState();
    renderV22SimState(simState);
  } catch (_error) {
    setV22SimMessage("시뮬 상태 조회 실패", "error");
  }
  await refreshGridQuotePrices();
  startGridQuoteAutoRefresh();
  try {
    await refreshState();
  } catch (error) {
    setLastEvent("초기 상태 조회 실패");
  }
  try {
    await refreshQuotePanel("005930", selectedRange);
  } catch (_error) {
    quoteResult.textContent = "기본 시세 조회 실패";
    renderCandles([]);
  }
  if (quoteSymbolCodeInput && quoteSymbolNameInput) {
    try {
      const matched = await resolveNameFromTicker(quoteSymbolCodeInput.value);
      if (matched && matched.name) {
        quoteSymbolNameInput.value = matched.name;
      }
    } catch (_error) {
      // ignore name resolve error on bootstrap
    }
  }
  connectSSE();
}

bootstrapTradePage();
