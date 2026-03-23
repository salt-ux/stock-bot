## 목적
- 시세 데이터(quote/candle)를 공통 모델로 제공하는 어댑터 계층입니다.

## 책임
- 외부 시세 API 호출
- (키움) WebSocket 실시간 수신 + REST 폴백
- 응답 정규화(`market.Quote`, `market.Candle`)
- 종목명/코드 조회용 심볼 리졸빙(로컬 카탈로그)
- 캐시 정책(TTL) 적용

## 비책임
- 매수/매도 판단(전략)
- 주문 실행(브로커 주문 API)

## 핵심 파일
- `internal/market/market_provider.go`
- `internal/market/market_service.go`
- `internal/market/market_symbol_lookup_resolver.go`
- `internal/market/mock/mock_market_provider.go`
- `internal/market/kiwoom/kiwoom_market_provider.go`
- `internal/market/kiwoom/kiwoom_websocket_quote_stream.go`

## 확장 포인트
- provider 추가(예: 다른 데이터 공급자)
- interval 확장(1m/5m/1h)
- 재시도/서킷브레이커 정책

## 변경 체크리스트
- `/market/quote`, `/market/candles`, `/market/symbols/search` 응답 스키마 유지
- 캐시 hit/miss 로직 회귀 테스트
- provider 전환(`MARKET_PROVIDER`) 시 동일 인터페이스 보장
