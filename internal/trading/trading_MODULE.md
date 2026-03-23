## 목적
- 실거래 없이 주문을 즉시 체결하는 페이퍼 브로커 실행기입니다.

## 책임
- 주문 시뮬레이션(BUY/SELL)
- 포지션/현금/실현손익 상태 업데이트
- 중복 주문 방지

## 비책임
- 실증권사 주문 송신
- 전략 시그널 생성

## 핵심 파일
- `internal/trading/service.go`
- `internal/trading/trading_types.go`

## 변경 체크리스트
- 평균단가/손익 계산 정확성
- 중복 주문 윈도우 동작
- `/paper/order`, `/paper/state` API 회귀 확인
