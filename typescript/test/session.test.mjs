import assert from 'node:assert/strict';
import {test} from 'node:test';
import {generateKeyPairSync, verify, X509Certificate} from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {mkdtempSync, writeFileSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createSessionRequest, createSession, ExternalAccessError} from '../dist/index.js';

function fixture(type = 'rsa') {
  const pair = type === 'rsa' ? generateKeyPairSync('rsa', {modulusLength: 2048})
    : generateKeyPairSync('ec', {namedCurve: 'prime256v1'});
  const privateKey = Buffer.from(pair.privateKey.export({format: 'pem', type: 'pkcs8'}));
  // Node의 pipe를 /dev/stdin 경로로 다시 열지 않는다. 합성 키만 전용 임시 디렉터리에 저장한다.
  const directory = mkdtempSync(join(tmpdir(), 'ncp-synthetic-test-'));
  try {
    const keyPath = join(directory, 'key.pem');
    writeFileSync(keyPath, privateKey, {mode: 0o600});
    const result = spawnSync('openssl', ['req', '-new', '-x509', '-key', keyPath, '-subj', '/CN=synthetic-test-only', '-days', '1']);
    assert.equal(result.status, 0, `OpenSSL fixture generation failed: ${result.stderr?.toString()}`);
    return {privateKey, certificate: result.stdout.toString(), trustAnchorNrn: 'synthetic-trust', profileNrn: 'synthetic-profile', roleNrn: 'synthetic-role'};
  } finally { rmSync(directory, {recursive: true, force: true}); }
}

for (const type of ['rsa', 'ec']) {
  test(`${type}: wire 형식 및 서명 검증, 변조 거부`, () => {
    const options = fixture(type), now = Date.now();
    const request = createSessionRequest(options, now);
    const cert = new X509Certificate(options.certificate);
    const algorithm = type === 'rsa' ? 'X509-RSA-SHA256' : 'X509-ECDSA-SHA256';
    assert.equal(request.headers['x-ncp-iam-x509-algorithm'], algorithm);
    assert.equal(request.headers['x-ncp-iam-x509'], cert.raw.toString('base64'));
    assert.deepEqual(JSON.parse(request.body), {durationSeconds: 600, trustAnchorNrn: options.trustAnchorNrn, profileNrn: options.profileNrn, roleNrn: options.roleNrn});
    const message = `POST /sessions\n${now}\n${algorithm}\n${cert.raw.toString('base64')}`;
    const signature = Buffer.from(request.headers['x-ncp-iam-signature-v2'], 'base64');
    assert.equal(verify('sha256', Buffer.from(message), cert.publicKey, signature), true);
    assert.equal(verify('sha256', Buffer.from(message + '!'), cert.publicKey, signature), false);
    options.privateKey.fill(0);
  });
}

test('일치하지 않는 키·만료·타임스탬프·입력 거부', () => {
  const a = fixture(), b = fixture();
  for (const [options, timestamp] of [
    [{...a, privateKey: b.privateKey}, Date.now()],
    [a, NaN], [a, -1], [a, Date.now() + 172800000], [a, 0],
    [{...a, roleNrn: ''}, Date.now()], [{...a, roleNrn: 'role\nsecret'}, Date.now()],
    [{...a, certificate: 'sensitive-invalid-input'}, Date.now()]
  ]) assert.throws(() => createSessionRequest(options, timestamp), error => error instanceof ExternalAccessError && error.code === 'invalid_input' && !error.message.includes('sensitive'));
});

test('세션 응답, 오류 비노출, 리다이렉트 설정, 자동 재시도 없음', async t => {
  const options = fixture();
  const credential = {accessKey: 'synthetic-access', keySecret: 'synthetic-secret', createTime: new Date().toISOString(), expireTime: new Date(Date.now() + 590000).toISOString()};
  let calls = 0;
  const stub = t.mock.method(globalThis, 'fetch', async (url, init) => {
    calls++;
    assert.equal(url, 'https://externalaccess.apigw.ntruss.com/auth/v1/sessions');
    assert.equal(init.redirect, 'error');
    assert.equal(init.method, 'POST');
    assert.ok(init.signal instanceof AbortSignal);
    return Response.json({roleNrn: options.roleNrn, credential}, {status: 201});
  });
  assert.deepEqual(await createSession(options), credential);
  assert.equal(calls, 1);
  for (const response of [
    new Response('sensitive-provider-error', {status: 403}),
    new Response('', {status: 302, headers: {location: 'https://invalid.example'}}),
    new Response('sensitive-malformed-json', {status: 201}),
    Response.json({roleNrn: 'wrong', credential}, {status: 201}),
    Response.json({roleNrn: options.roleNrn, credential: {...credential, expireTime: '2000-01-01T00:00:00Z'}}, {status: 201}),
    Response.json({roleNrn: options.roleNrn, credential: {...credential, accessKey: ''}}, {status: 201}),
    Response.json({roleNrn: options.roleNrn, credential, padding: 'x'.repeat(16384)}, {status: 201})
  ]) {
    let attempts = 0;
    stub.mock.mockImplementation(async () => { attempts++; return response; });
    await assert.rejects(createSession(options), error => error instanceof ExternalAccessError && !error.message.includes('sensitive'));
    assert.equal(attempts, 1);
  }
  stub.mock.mockImplementation(async () => { throw new Error('sensitive-network-error'); });
  await assert.rejects(createSession(options), error => error.code === 'request_failed' && !error.message.includes('sensitive'));
});
