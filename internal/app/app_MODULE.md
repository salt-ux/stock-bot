## 목적
- 애플리케이션 조립(bootstrap) 계층의 자리입니다.

## 현재 상태
- 아직 실구현 없음(placeholder)

## 향후 책임
- 모듈 초기화 순서 관리
- graceful shutdown 관리
- 의존성 주입(services/repositories)

## 변경 체크리스트
- 초기화 로직이 `cmd/api/main.go`와 중복되지 않도록 이동 계획 유지
