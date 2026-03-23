## 목적
- 사용자 인증(회원가입/로그인) 로직을 담당합니다.

## 책임
- `id/password` 유효성 검사
- 인증 저장소(MySQL) 읽기/쓰기
- 인증 실패/성공 에러 표준화

## 비책임
- 세션/JWT 발급
- 권한(Role) 관리

## 핵심 파일
- `internal/auth/auth_store.go`
- `internal/auth/mysql_store.go`

## 확장 포인트
- 비밀번호 해시(`bcrypt`) 도입
- 로그인 시도 제한(브루트포스 방지)
- 세션/토큰 발급 연동

## 변경 체크리스트
- `Register`/`Authenticate` 동작 유지 여부
- 비밀번호 정책(최소 길이) 회귀 테스트
- DB 스키마(`users`) 호환성 확인
