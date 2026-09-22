import {constants, createPrivateKey, sign, X509Certificate} from 'node:crypto';

export interface SessionOptions {
  certificate: string;
  privateKey: Buffer;
  trustAnchorNrn: string;
  profileNrn: string;
  roleNrn: string;
}

export interface Credential {
  accessKey: string;
  keySecret: string;
  createTime: string;
  expireTime: string;
}

export class ExternalAccessError extends Error {
  constructor(readonly code: 'invalid_input' | 'request_failed' | 'invalid_response') {
    super(`NCP External Access: ${code}`);
    this.name = 'ExternalAccessError';
  }
}

/** 서명용 요청만 구성한다. 입력·암호 라이브러리의 원문 오류는 공개하지 않는다. */
export function createSessionRequest(options: SessionOptions, timestamp = Date.now()) {
  try {
    if (!Number.isSafeInteger(timestamp) || timestamp < 0) throw new Error();
    for (const nrn of [options.trustAnchorNrn, options.profileNrn, options.roleNrn]) {
      if (typeof nrn !== 'string' || !nrn || nrn.length > 2048 || /[\s\x00-\x1f\x7f]/.test(nrn)) throw new Error();
    }
    const certificate = new X509Certificate(options.certificate);
    const key = createPrivateKey(options.privateKey);
    if (!certificate.checkPrivateKey(key) || timestamp < Date.parse(certificate.validFrom)
      || timestamp >= Date.parse(certificate.validTo)) throw new Error();
    const keyType = certificate.publicKey.asymmetricKeyType;
    const algorithm = keyType === 'rsa' ? 'X509-RSA-SHA256'
      : keyType === 'ec' ? 'X509-ECDSA-SHA256' : undefined;
    if (!algorithm) throw new Error();
    const der = certificate.raw.toString('base64');
    const signature = sign('sha256', Buffer.from(`POST /sessions\n${timestamp}\n${algorithm}\n${der}`), {
      key, ...(keyType === 'rsa' ? {padding: constants.RSA_PKCS1_PADDING} : {dsaEncoding: 'der' as const})
    }).toString('base64');
    return {
      method: 'POST' as const,
      headers: {
        'content-type': 'application/json',
        'x-ncp-iam-timestamp': String(timestamp),
        'x-ncp-iam-x509-algorithm': algorithm,
        'x-ncp-iam-x509': der,
        'x-ncp-iam-signature-v2': signature
      },
      body: JSON.stringify({durationSeconds: 600, trustAnchorNrn: options.trustAnchorNrn,
        profileNrn: options.profileNrn, roleNrn: options.roleNrn})
    };
  } catch { throw new ExternalAccessError('invalid_input'); }
}

/** 자동 재시도·캐시 없이 세션 하나를 요청한다. 인증정보의 보관·갱신은 호출자가 소유한다. */
export async function createSession(options: SessionOptions): Promise<Credential> {
  const signed = createSessionRequest(options);
  let raw: Buffer | undefined;
  const chunks: Buffer[] = [];
  try {
    const response = await fetch('https://externalaccess.apigw.ntruss.com/auth/v1/sessions', {
      ...signed, redirect: 'error', signal: AbortSignal.timeout(20000)
    });
    if (response.status !== 201 || !response.body) {
      await response.body?.cancel();
      throw new ExternalAccessError('request_failed');
    }
    const reader = response.body.getReader();
    let size = 0;
    try {
      for (;;) {
        const next = await reader.read();
        if (next.done) break;
        size += next.value.byteLength;
        if (size > 16384) throw new ExternalAccessError('invalid_response');
        chunks.push(Buffer.from(next.value));
      }
    } finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
    raw = Buffer.concat(chunks);
    let body;
    try { body = JSON.parse(raw.toString('utf8')); }
    catch { throw new ExternalAccessError('invalid_response'); }
    const c = body?.credential;
    const now = Date.now();
    if (body?.roleNrn !== options.roleNrn || !c || Array.isArray(c)
      || typeof c.accessKey !== 'string' || !/^[A-Za-z0-9_-]{1,512}$/.test(c.accessKey)
      || typeof c.keySecret !== 'string' || !c.keySecret || c.keySecret.length > 4096
      || typeof c.createTime !== 'string' || !Number.isFinite(Date.parse(c.createTime))
      || typeof c.expireTime !== 'string' || !Number.isFinite(Date.parse(c.expireTime))
      || Date.parse(c.createTime) > now + 5000 || Date.parse(c.expireTime) <= now
      || Date.parse(c.expireTime) <= Date.parse(c.createTime)
      || Date.parse(c.expireTime) > now + 605000) throw new ExternalAccessError('invalid_response');
    return {accessKey: c.accessKey, keySecret: c.keySecret, createTime: c.createTime, expireTime: c.expireTime};
  } catch (error) {
    if (error instanceof ExternalAccessError) throw error;
    throw new ExternalAccessError('request_failed');
  } finally { raw?.fill(0); for (const chunk of chunks) chunk.fill(0); }
}
