## 목적
- 브로커 구현체를 선택/생성하는 팩토리 계층입니다.

## 책임
- 설정값 기반 provider 선택(`kiwoom` 등)
- 상위 계층이 구현체에 직접 의존하지 않도록 분리

## 비책임
- 브로커 API 호출 로직

## 핵심 파일
- `internal/brokers/brokers_factory.go`

## 확장 포인트
- 신규 브로커 provider 추가

## 변경 체크리스트
- `BROKER_PROVIDER` 검증값과 factory 분기 동기화
