# scheduler module

## 책임
- `robfig/cron` 기반 스케줄 작업 등록/실행
- 장 시작 이벤트 로그 작업
- 전략 시그널을 기반으로 페이퍼 주문 자동 실행 작업

## 주요 파일
- `internal/scheduler/cron_scheduler_runner.go`
- `internal/scheduler/cron_scheduler_runner_test.go`

## 외부 의존
- `internal/strategy` 엔진
- `internal/trading` 페이퍼 주문 서비스
- `github.com/robfig/cron/v3`

## 제공 기능
- `NewCronSchedulerRunner`: 크론 작업 생성/검증
- `Start` / `Stop`: 앱 수명주기 연동
- `Snapshot`: 현재 스케줄 상태 조회(다음 실행 시각, 마지막 실행 결과)
