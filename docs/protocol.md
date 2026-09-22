# 초기 프로토콜 범위와 출처

## 요청

- URL: `https://externalaccess.apigw.ntruss.com/auth/v1/sessions`
- Method: `POST`, Content-Type: `application/json`
- Body: `durationSeconds: 600`, `trustAnchorNrn`, `profileNrn`, `roleNrn`
- Timestamp: Unix epoch 밀리초의 십진수 문자열
- Certificate: leaf X.509 DER의 표준 Base64
- Algorithm: RSA는 `X509-RSA-SHA256`, EC는 `X509-ECDSA-SHA256`

서명 입력은 다음 UTF-8 문자열이며 마지막 개행은 없습니다. 요청 URL의 `/auth/v1` prefix는 서명 경로에 포함하지 않습니다.

```text
POST /sessions\n{timestamp}\n{algorithm}\n{base64CertificateDer}
```

SHA-256, RSA PKCS#1 v1.5 또는 ASN.1 DER ECDSA 서명을 표준 Base64로 보냅니다. 헤더는 `x-ncp-iam-timestamp`, `x-ncp-iam-x509-algorithm`, `x-ncp-iam-x509`, `x-ncp-iam-signature-v2`입니다.

## 응답과 실패

HTTP 201만 수용합니다. JSON `roleNrn`이 요청과 같아야 하며 `credential`의 `accessKey`, `keySecret`, `createTime`, `expireTime`을 반환합니다. 추가 응답 필드는 반환하지 않습니다. 만료된 credential, 600초 요청보다 과도하게 긴 미래 만료(5초 허용), 잘못된 역할·필드·과대 응답은 거부합니다.

20초 timeout, redirect 거부, 최대 16 KiB 응답을 적용합니다. 자동 retry/cache는 없습니다. 이미 작성한 요청을 오래 저장하거나 재전송하지 마세요. `createSessionRequest`/`CreateSessionRequest`의 반환값에는 인증 관련 헤더가 있으므로 로그에 남기지 마세요.

Go의 기본 HTTP transport는 표준 환경 proxy 설정을 따를 수 있습니다. Node의 proxy 사용 여부도 실행 환경에 달려 있습니다. 양쪽 모두 OS/런타임 신뢰 저장소를 사용하며 전역 transport를 변경한 경우 보안 책임은 호출자에게 있습니다.

## 출처와 검증 수준

[공식 helper 안내](https://guide.ncloud-docs.com/docs/en/external-access-signing-helper-cli)는 서비스 이용 흐름·입력·출력의 참고입니다. 위 wire 세부사항 전체가 공개된 안정 API 명세라고 주장하지 않습니다.

기반 Node 구현은 공개 자료·합성 요청 관찰·공식 바이너리의 호환 관련 함수 및 어셈블리 분석을 참고해 표준 암호 API로 작성했습니다. 엄격한 클린룸 구현이라고 주장하지 않습니다. 공식 바이너리·아카이브·제3자 소스는 이 저장소에 포함하지 않습니다. NAVER Cloud의 승인이나 공개 조건 검토가 완료됐다는 의미는 아닙니다.

기반 구현의 개발 실환경 검증과 이 저장소의 합성 테스트를 구분합니다. Go 구현, EC 실환경, 중간 인증서·다른 endpoint·여러 OS는 별도의 검증이 필요합니다. 인증서 체인과 신원 정책은 NCP가 검증하며 이 라이브러리는 CA가 아닙니다.
