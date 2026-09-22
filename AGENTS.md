# 작업 지침

- 이 저장소는 비공식 NCP External Access 클라이언트 라이브러리다. CLI·바이너리·메일·KMS·내부 서비스 인증은 범위 밖이다.
- 수정 전 README, docs/protocol.md, 해당 언어의 구현과 테스트를 읽는다.
- TypeScript와 Go의 wire 형식은 같게 유지하고, 변경 시 양쪽 테스트를 실행한다.
- 실제 인증서·개인키·임시 credential·고객 식별자를 저장소, 테스트, 로그에 넣지 않는다. 테스트는 합성 키만 사용한다.
- 실제 NCP 요청, GitHub push, 패키지 발행은 별도 명시 요청이 있어야 한다.
- 오류에 provider 원문이나 credential을 포함하지 않는다. TLS 검증 생략·리다이렉트·자동 재시도·임의 endpoint를 추가하지 않는다.
- 런타임 의존성 추가, 공개 API 확대, 지원 endpoint 확대는 이유를 먼저 설명한다.
- 로컬 검증: `cd typescript && npm ci && npm test && npm pack --dry-run`, `cd go && go test -race ./... && go vet ./...`, `git diff --check`.
- Git 커밋은 한국어 Conventional Commits를 사용한다. AI 기여 시 확인한 모델과 작업 ID를 AI-Tool/AI-Model/AI-Task-ID trailer에 기록한다. 알 수 없는 값은 추정하지 않는다.
