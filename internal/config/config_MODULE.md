## 목적
- 애플리케이션 환경설정 로딩과 검증을 담당합니다.

## 책임
- 환경변수 파싱/기본값 설정
- 설정 유효성 검증(포트, provider, 리스크 값)
- 하위 모듈에서 사용할 `Config` 구조 제공

## 비책임
- 실제 비즈니스 로직 수행
- 외부 API 호출

## 핵심 파일
- `internal/config/config.go`
- `internal/config/config_test.go`

## 확장 포인트
- 신규 모듈 설정 추가
- 환경별 설정 분기(local/stage/prod)

## 변경 체크리스트
- `.env.example`와 키 동기화
- 기본값/검증 로직 테스트 추가
