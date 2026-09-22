# externalaccess

Go 1.26.6+용 비공식 NCP External Access 클라이언트. MIT, 표준 라이브러리만 사용합니다. CLI·빌드된 바이너리는 제공하지 않습니다. 아직 버전 태그가 없습니다.

[저장소 README](https://github.com/dreamyoungs/ncp-external-access#readme)의 제작 배경·NAVER Cloud 미승인·공개 중단 가능성을 확인하세요.

## API

```go
import (
    "context"
    externalaccess "github.com/dreamyoungs/ncp-external-access/go"
)

// certificatePEM과 privateKeyPEM은 호출자가 안전하게 읽거나 복호화한 []byte입니다.
credential, err := externalaccess.CreateSession(context.Background(), externalaccess.SessionOptions{
    Certificate: certificatePEM,
    PrivateKey: privateKeyPEM,
    TrustAnchorNrn: "YOUR_TRUST_ANCHOR_NRN",
    ProfileNrn: "YOUR_PROFILE_NRN",
    RoleNrn: "YOUR_ROLE_NRN",
})
// err를 확인한 뒤 credential을 사용하세요. 키·반환값을 로그에 출력하지 마세요.
```

`CreateSessionRequest(options, time.Now())`는 네트워크 없이 서명된 `*http.Request`를 만듭니다. `CreateSession(ctx, options)`는 20초 timeout과 호출자 context를 적용해 한 번 요청합니다. 실패는 `ErrInvalidInput`, `ErrRequestFailed`, `ErrInvalidResponse`로 구분합니다.

단일 PEM leaf 인증서와 암호화되지 않은 PKCS#8/PKCS#1/SEC1 RSA·EC 개인키를 지원합니다. 고정 Public endpoint에 600초 세션을 요청합니다. 인증서 갱신·키 보관·세션 캐시·재시도는 호출자가 담당합니다. Go 실환경 호환 검증은 아직 수행하지 않았습니다.

```sh
go test -race ./...
go vet ./...
```
