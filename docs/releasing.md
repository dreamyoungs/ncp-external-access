# 릴리스 절차

2026-09-22 `@dreamyoungs/ncp-external-access@0.1.0`의 npm 발행 명령이 성공했습니다. 직후 공개 조회는 404여서 설치 가능 여부는 별도 확인 대기입니다. Go tag는 아직 없으며 CLI·실행 파일은 배포하지 않습니다.

## 최초 공개 전

1. README의 비공식 상태·공개 중단 안내와 출처·라이선스 적용 범위를 검토합니다.
2. npm scope 소유권·패키지명·발행 권한을 확인합니다. 최초 발행에서는 `dreamyoungs` 계정과 해당 scope의 owner 권한을 확인했습니다.
3. GitHub의 비공개 취약점 제보, branch protection, 필요한 리뷰 정책을 설정합니다. 로컬 YAML만으로 서버 설정이 적용되지는 않습니다.
4. 양쪽 테스트와 package contents 검사를 실행하고 실제 비밀값이 없는지 검토합니다. TypeScript 디렉터리에서 `npm audit --audit-level=low`, Go 디렉터리에서 `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -show verbose ./...`를 실행합니다. 실패 또는 취약점 발견 시 발행을 중단하고 원인을 확인합니다.
5. 분리 패키지의 실환경 검증은 별도 승인 후 최소 권한으로 수행하고 결과를 기록합니다. 기존 앱의 성공을 Go 실환경 성공으로 표시하지 않습니다.

## TypeScript

`typescript/package.json` 버전을 변경하고 `npm ci`, `npm test`, `npm pack --dry-run`을 실행합니다. tarball에는 `dist`, README, LICENSE, package.json만 포함하는지 확인합니다. 이 저장소는 자동 publish workflow를 제공하지 않습니다. npm 발행은 승인된 담당자가 수행합니다. 태그는 `typescript/v0.1.0`처럼 언어별로 구분합니다.

## Go

`go test -race ./...`, `go vet ./...`, `gofmt` 검사 후 `go/v0.1.0` 형태로 태그합니다. 하위 디렉터리 모듈 경로는 `github.com/dreamyoungs/ncp-external-access/go`입니다. 바이너리 릴리스가 아니라 소스 모듈입니다.

두 언어의 버전은 독립적입니다. 루트 LICENSE가 정본이며 npm 패키지의 LICENSE와 Go 하위 모듈의 LICENSE를 동일하게 유지합니다. Go는 모듈 zip에도 라이선스가 포함되도록 모듈 안에 사본을 둡니다.
