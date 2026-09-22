// Package externalaccess는 비공식 NCP External Access X.509 클라이언트다.
package externalaccess

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidInput    = errors.New("NCP External Access: invalid_input")
	ErrRequestFailed   = errors.New("NCP External Access: request_failed")
	ErrInvalidResponse = errors.New("NCP External Access: invalid_response")
)

type SessionOptions struct {
	Certificate    []byte
	PrivateKey     []byte
	TrustAnchorNrn string
	ProfileNrn     string
	RoleNrn        string
}

type Credential struct {
	AccessKey  string `json:"accessKey"`
	KeySecret  string `json:"keySecret"`
	CreateTime string `json:"createTime"`
	ExpireTime string `json:"expireTime"`
}

// CreateSessionRequest는 네트워크 요청 없이 서명한다. 개인키 PEM은 암호화되지 않은 PKCS#8, PKCS#1, SEC1을 지원한다.
func CreateSessionRequest(options SessionOptions, now time.Time) (*http.Request, error) {
	stamp := now.UnixMilli()
	if stamp < 0 || stamp > 9007199254740991 {
		return nil, ErrInvalidInput
	}
	for _, nrn := range []string{options.TrustAnchorNrn, options.ProfileNrn, options.RoleNrn} {
		if len(nrn) == 0 || len(nrn) > 2048 || strings.ContainsFunc(nrn, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
			return nil, ErrInvalidInput
		}
	}
	block, _ := pem.Decode(options.Certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, ErrInvalidInput
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil || now.Before(cert.NotBefore) || !now.Before(cert.NotAfter) {
		return nil, ErrInvalidInput
	}
	keyBlock, _ := pem.Decode(options.PrivateKey)
	if keyBlock == nil {
		return nil, ErrInvalidInput
	}
	var parsed any
	switch keyBlock.Type {
	case "PRIVATE KEY":
		parsed, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	case "RSA PRIVATE KEY":
		parsed, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		parsed, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	default:
		return nil, ErrInvalidInput
	}
	if err != nil {
		return nil, ErrInvalidInput
	}
	key, ok := parsed.(crypto.Signer)
	if !ok {
		return nil, ErrInvalidInput
	}
	pub, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil || !bytes.Equal(pub, cert.RawSubjectPublicKeyInfo) {
		return nil, ErrInvalidInput
	}
	var algorithm string
	switch key.(type) {
	case *rsa.PrivateKey:
		algorithm = "X509-RSA-SHA256"
	case *ecdsa.PrivateKey:
		algorithm = "X509-ECDSA-SHA256"
	default:
		return nil, ErrInvalidInput
	}
	der := base64.StdEncoding.EncodeToString(cert.Raw)
	stampText := fmt.Sprint(stamp)
	hash := sha256.Sum256([]byte("POST /sessions\n" + stampText + "\n" + algorithm + "\n" + der))
	signature, err := key.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return nil, ErrInvalidInput
	}
	body, err := json.Marshal(struct {
		DurationSeconds int    `json:"durationSeconds"`
		TrustAnchorNrn  string `json:"trustAnchorNrn"`
		ProfileNrn      string `json:"profileNrn"`
		RoleNrn         string `json:"roleNrn"`
	}{600, options.TrustAnchorNrn, options.ProfileNrn, options.RoleNrn})
	if err != nil {
		return nil, ErrInvalidInput
	}
	req, err := http.NewRequest(http.MethodPost, "https://externalaccess.apigw.ntruss.com/auth/v1/sessions", bytes.NewReader(body))
	if err != nil {
		return nil, ErrInvalidInput
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ncp-iam-timestamp", stampText)
	req.Header.Set("x-ncp-iam-x509-algorithm", algorithm)
	req.Header.Set("x-ncp-iam-x509", der)
	req.Header.Set("x-ncp-iam-signature-v2", base64.StdEncoding.EncodeToString(signature))
	return req, nil
}

var client = &http.Client{
	Timeout:       20 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// CreateSession은 캐시·자동 재시도 없이 요청한다. 호출자가 취소와 credential 갱신을 관리한다.
func CreateSession(ctx context.Context, options SessionOptions) (Credential, error) {
	req, err := CreateSessionRequest(options, time.Now())
	if err != nil {
		return Credential{}, err
	}
	response, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return Credential{}, ErrRequestFailed
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return Credential{}, ErrRequestFailed
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 16385))
	defer clear(raw)
	if err != nil {
		return Credential{}, ErrRequestFailed
	}
	if len(raw) > 16384 {
		return Credential{}, ErrInvalidResponse
	}
	var body struct {
		RoleNrn    string     `json:"roleNrn"`
		Credential Credential `json:"credential"`
	}
	if json.Unmarshal(raw, &body) != nil || body.RoleNrn != options.RoleNrn {
		return Credential{}, ErrInvalidResponse
	}
	c := body.Credential
	created, err1 := time.Parse(time.RFC3339Nano, c.CreateTime)
	expires, err2 := time.Parse(time.RFC3339Nano, c.ExpireTime)
	now := time.Now()
	if len(c.AccessKey) == 0 || len(c.AccessKey) > 512 || strings.ContainsFunc(c.AccessKey, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-')
	}) || len(c.KeySecret) == 0 || len(c.KeySecret) > 4096 || err1 != nil || err2 != nil || created.After(now.Add(5*time.Second)) || !expires.After(now) || !expires.After(created) || expires.After(now.Add(605*time.Second)) {
		return Credential{}, ErrInvalidResponse
	}
	return c, nil
}
