# ncp-external-access

NCP External Access의 X.509 인증을 사용하는 **비공식 TypeScript·Go 클라이언트 라이브러리**입니다. 사용자가 제공한 인증서와 개인키로 요청을 서명하고 NCP가 발급하는 임시 자격 증명을 받습니다.

## 왜 만들었나요?

인증서 기반 External Access를 선택한 이유는 보안이었습니다. 그러나 사용하던 공식 Signing Helper 배포본에서 구형 Go 런타임 관련 알려진 취약점 경고가 검출됐습니다. 경고가 곧 실제 악용 가능성을 뜻하지는 않지만, 보안 검토·예외 승인·공급자 업데이트 대기를 반복해야 하는 운영 부담이 있었습니다.

그래서 표준 암호 라이브러리를 이용해 필요한 인증 경로를 직접 구현했습니다. 애플리케이션 개발자가 소스와 런타임의 업데이트를 직접 관리하면서 NCP의 기존 인증·권한 체계를 계속 사용할 수 있도록 하는 것이 목적입니다. 인증 우회나 NCP 서비스 대체를 위한 프로젝트가 아닙니다. 이 설명은 당시 사용한 배포본에 관한 것이며, 현재의 모든 공식 배포본에 동일한 문제가 있다는 주장이 아닙니다.

## 비공식 프로젝트 및 공개 유지 안내

**이 프로젝트는 NAVER Cloud의 승인·공인·후원을 받지 않았습니다.** NAVER Cloud와 제휴된 프로젝트가 아니며 공식 SDK 또는 공식 Signing Helper의 완전한 호환 대체품을 표방하지 않습니다. NCP 관련 명칭은 호환 대상 서비스를 설명하기 위해 사용합니다.

보안상의 필요로 작성한 구현을 공유하는 프로젝트입니다. 향후 NAVER Cloud로부터 연락이나 요청을 받는 경우, 그 내용에 따라 구현·배포 방침을 변경하거나 **저장소를 비공개로 전환하고, 공개를 중단하거나, 저장소를 삭제할 수 있습니다.** 지속적인 공개·지원은 보장하지 않습니다.

저장소의 향후 공개 여부와 이미 적법하게 MIT 라이선스로 제공된 사본의 이용 허락은 별개입니다. 저장소 삭제만으로 기존 사본의 MIT 이용 허락이 철회된다는 의미는 아닙니다. 이 프로젝트의 라이선스는 NCP 서비스 약관이나 제3자의 권리를 대신하지 않습니다.

## 범위와 현재 상태

| 패키지 | 형태 | 상태 |
| --- | --- | --- |
| [TypeScript](typescript/README.md) | Node.js ESM·타입 선언, npm 패키지 | 0.1.0 발행 명령 성공, 공개 조회 확인 대기 |
| [Go](go/README.md) | Go 모듈 소스 | 초기 구현, 미태그 |

- RSA PKCS#1 v1.5 / ECDSA SHA-256 서명, 단일 PEM 인증서 및 암호화되지 않은 PEM 개인키 입력
- Public endpoint의 CreateSession 요청, 600초 세션 요청
- 인증서·키 일치와 유효기간 확인, 20초 timeout, 16 KiB 응답 상한, 응답 역할·credential 형식 확인
- CLI·바이너리 배포·Rust 구현·KMS·메일 발송·캐시·자동 갱신·자동 재시도는 포함하지 않습니다.
- 중간 인증서 전달, Gov/금융 endpoint, 공식 helper의 전체 옵션은 지원하지 않습니다.
- 서버용입니다. 개인키를 브라우저나 프런트엔드 번들에 포함하지 마세요.

기반이 된 Node 구현은 2026-09-22 개발 환경에서 공식 helper와의 합성 RSA 요청 비교, 실제 임시 인증정보 발급 및 이를 이용한 메일 수신까지 확인했습니다. **이 저장소로 분리한 패키지와 Go 구현은 별도의 구현입니다.** 로컬 합성 테스트가 실제 NCP 상호운용 검증·보안 감사·모든 환경의 지원을 대신하지 않습니다.

## 개발

Node.js 24 이상, npm, Go 1.26.6 이상 및 TypeScript 테스트용 OpenSSL이 필요합니다. 런타임 외부 패키지 의존성은 없습니다. TypeScript 빌드에만 TypeScript와 Node 타입 선언을 사용합니다.

```sh
cd typescript
npm ci
npm test
npm pack --dry-run
```

```sh
cd go
go test -race ./...
go vet ./...
```

실제 NCP 계정 없이 로컬 합성 키와 응답으로 검사합니다. [프로토콜·검증 범위](docs/protocol.md), [기여 안내](CONTRIBUTING.md), [보안 정책](SECURITY.md), [릴리스 절차](docs/releasing.md)를 참고하세요.

## 라이선스

[MIT](LICENSE). 상업적 이용·수정·재배포가 가능하며 저작권·허가 고지를 유지해야 합니다. 무보증으로 제공합니다.

## 공식 자료

- [External Access 인증 및 권한 관리](https://guide.ncloud-docs.com/docs/subaccount-external-access)
- [External Access Signing Helper CLI](https://guide.ncloud-docs.com/docs/en/external-access-signing-helper-cli)
