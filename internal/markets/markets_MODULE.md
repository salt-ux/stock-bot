## 목적
- 시세 provider 구현체를 선택/생성하는 팩토리 계층입니다.

## 책임
- 설정값 기반 market provider 선택(`mock`, `kiwoom`)
- market service 초기화 위임

## 비책임
- 시세 파싱/캐시 세부 로직

## 핵심 파일
- `internal/markets/markets_factory.go`
- `internal/markets/factory_test.go`

## 확장 포인트
- 신규 market provider 분기 추가

## 변경 체크리스트
- `MARKET_PROVIDER` 검증값과 factory 분기 동기화
