## 목적
- DB 접근을 `sqlc` 기반으로 캡슐화한 영속 계층입니다.

## 책임
- 테이블별 CRUD/조회 쿼리 래핑
- 트랜잭션 경계 제공(`Store.InTx`)
- 상위 계층에 안정적인 저장소 인터페이스 제공

## 비책임
- HTTP 요청 처리
- 전략 판단 로직

## 핵심 파일
- `internal/repository/repository_store.go`
- `internal/repository/users.go`
- `internal/repository/portfolios.go`
- `internal/repository/orders.go`
- `internal/repository/sqlc/*`

## 확장 포인트
- 쿼리 최적화/인덱스 반영
- 페이징/필터 조회 API 추가
- 복합 트랜잭션 유스케이스 확장

## 변경 체크리스트
- `sqlc generate` 후 빌드 통과 여부
- `ErrNotFound` 처리 일관성
- 스키마 변경 시 migration/query/sqlc 동기화
