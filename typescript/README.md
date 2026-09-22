# @dreamyoungs/ncp-external-access

Node.js 24+용 비공식 NCP External Access 클라이언트. MIT, ESM, 런타임 외부 의존성 없음.

공개 배경·NAVER Cloud 미승인·공개 중단 가능성은 [저장소 README](https://github.com/dreamyoungs/ncp-external-access#readme)를 확인하세요.

사용하던 공식 helper 배포본의 구형 런타임 관련 취약점 경고와 업데이트 대기 부담을 줄이고, 개발자가 소스·런타임을 직접 관리하기 위해 만들었습니다. 현재의 모든 공식 배포본에 동일한 문제가 있다는 뜻은 아닙니다.

NAVER Cloud의 승인·공인·후원을 받지 않았습니다. 향후 연락이나 요청 내용에 따라 공개 중단·비공개 전환·저장소 삭제가 있을 수 있습니다. 이는 이미 적법하게 MIT로 제공된 사본의 이용 허락을 철회한다는 의미는 아닙니다. 이 패키지 자체의 검증은 로컬 합성 테스트이며, 기반 앱의 실환경 성공을 분리 패키지의 실환경 검증으로 간주하지 않습니다.

## 설치

```sh
npm install @dreamyoungs/ncp-external-access
```

## 사용 예시

인증서·개인키는 호출자가 안전하게 읽거나 복호화해 전달합니다.

```ts
import {readFile} from 'node:fs/promises';
import {createSession} from '@dreamyoungs/ncp-external-access';

const privateKey = await readFile('/secure/private-key.pem');
try {
  const credential = await createSession({
    certificate: await readFile('/secure/certificate.pem', 'utf8'),
    privateKey,
    trustAnchorNrn: 'YOUR_TRUST_ANCHOR_NRN',
    profileNrn: 'YOUR_PROFILE_NRN',
    roleNrn: 'YOUR_ROLE_NRN'
  });
  // credential.accessKey/keySecret을 NCP API 서명에 사용하세요. 로그에 출력하지 마세요.
  // credential.expireTime 이전에 갱신하고 인증서 만료시간도 고려하세요.
} finally {
  privateKey.fill(0); // 다른 문자열·런타임 복사본의 완전 삭제를 보장하지 않습니다.
}
```

`createSessionRequest(options, timestamp?)`는 네트워크 없이 서명된 요청을 구성합니다. `createSession(options)`는 고정 Public endpoint에 한 번 요청합니다. 실패 시 `ExternalAccessError.code`는 `invalid_input`, `request_failed`, `invalid_response` 중 하나이며 원문 provider 오류를 포함하지 않습니다.

단일 PEM leaf 인증서, 암호화되지 않은 PEM RSA/EC 개인키, 600초 요청만 지원합니다. 캐시·갱신·재시도·CLI·KMS는 없습니다. 브라우저에서 사용하지 마세요. 인증서·키·서명 요청·반환 credential을 기록하지 마세요.

```sh
npm ci
npm test
npm pack --dry-run
```

테스트는 OpenSSL로 합성 인증서를 생성하며 NCP를 호출하지 않습니다.
