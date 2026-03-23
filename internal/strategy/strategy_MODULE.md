## 목적
- 캔들 데이터를 기반으로 매매 시그널(BUY/SELL/HOLD)을 생성합니다.

## 책임
- 전략 인터페이스 정의
- 전략 실행 엔진 제공
- 전략별 시그널/메타데이터 산출

## 비책임
- 주문 체결 처리
- 리스크 한도 집행

## 핵심 파일
- `internal/strategy/strategy_contract.go`
- `internal/strategy/strategy_types.go`
- `internal/strategy/engine.go`
- `internal/strategy/sma/crossover.go`
- `internal/strategy/infinitebuy/infinitebuy_strategy.go`

## 확장 포인트
- 신규 전략 추가(폴더 분리)
- 전략 파라미터 저장/버전관리
- 백테스트 엔진 연결

## 변경 체크리스트
- 시그널 타입(BUY/SELL/HOLD) 호환성
- 전략별 테스트 케이스(매수/매도/홀드)
- `/strategy/signal` 파라미터 회귀 확인
