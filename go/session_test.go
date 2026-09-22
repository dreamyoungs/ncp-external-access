package externalaccess

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, ec bool) SessionOptions {
	t.Helper()
	var key crypto.Signer
	var err error
	if ec {
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	} else {
		key, err = rsa.GenerateKey(rand.Reader, 2048)
	}
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "synthetic-test-only"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, key.Public(), key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return SessionOptions{Certificate: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), PrivateKey: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), TrustAnchorNrn: "synthetic-trust", ProfileNrn: "synthetic-profile", RoleNrn: "synthetic-role"}
}

func TestSignature(t *testing.T) {
	for _, ec := range []bool{false, true} {
		t.Run(map[bool]string{false: "rsa", true: "ec"}[ec], func(t *testing.T) {
			o := fixture(t, ec)
			r, err := CreateSessionRequest(o, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if r.Method != "POST" || r.URL.String() != "https://externalaccess.apigw.ntruss.com/auth/v1/sessions" {
				t.Fatal("wrong endpoint")
			}
			var body map[string]any
			if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 4 || body["durationSeconds"] != float64(600) || body["roleNrn"] != o.RoleNrn {
				t.Fatal("wrong body")
			}
			block, _ := pem.Decode(o.Certificate)
			cert, _ := x509.ParseCertificate(block.Bytes)
			algorithm := "X509-RSA-SHA256"
			if ec {
				algorithm = "X509-ECDSA-SHA256"
			}
			if r.Header.Get("x-ncp-iam-x509-algorithm") != algorithm || r.Header.Get("x-ncp-iam-x509") != base64.StdEncoding.EncodeToString(cert.Raw) {
				t.Fatal("wrong headers")
			}
			message := "POST /sessions\n" + r.Header.Get("x-ncp-iam-timestamp") + "\n" + algorithm + "\n" + r.Header.Get("x-ncp-iam-x509")
			sig, _ := base64.StdEncoding.DecodeString(r.Header.Get("x-ncp-iam-signature-v2"))
			for _, tamper := range []bool{false, true} {
				input := message
				if tamper {
					input += "!"
				}
				hash := sha256.Sum256([]byte(input))
				valid := false
				switch pub := cert.PublicKey.(type) {
				case *rsa.PublicKey:
					valid = rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash[:], sig) == nil
				case *ecdsa.PublicKey:
					valid = ecdsa.VerifyASN1(pub, hash[:], sig)
				}
				if valid == tamper {
					t.Fatal("signature verification mismatch")
				}
			}
		})
	}
}

func TestInvalidInput(t *testing.T) {
	o := fixture(t, false)
	other := fixture(t, false)
	badKey := o
	badKey.PrivateKey = other.PrivateKey
	badRole := o
	badRole.RoleNrn = "role\nsecret"
	for _, candidate := range []SessionOptions{badKey, badRole, {}} {
		if _, err := CreateSessionRequest(candidate, time.Now()); !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid input accepted")
		}
	}
	for _, now := range []time.Time{time.UnixMilli(-1), time.Now().Add(2 * time.Hour), time.UnixMilli(0)} {
		if _, err := CreateSessionRequest(o, now); !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid time accepted")
		}
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSession(t *testing.T) {
	o := fixture(t, false)
	original := client
	t.Cleanup(func() { client = original })
	c := Credential{AccessKey: "synthetic-access", KeySecret: "synthetic-secret", CreateTime: time.Now().UTC().Format(time.RFC3339), ExpireTime: time.Now().Add(590 * time.Second).UTC().Format(time.RFC3339)}
	valid, _ := json.Marshal(map[string]any{"roleNrn": o.RoleNrn, "credential": c})
	expired := c
	expired.ExpireTime = "2000-01-01T00:00:00Z"
	badExpiry, _ := json.Marshal(map[string]any{"roleNrn": o.RoleNrn, "credential": expired})
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"success", 201, string(valid), nil},
		{"denied", 403, "sensitive-provider-error", ErrRequestFailed},
		{"redirect", 302, "", ErrRequestFailed},
		{"malformed", 201, "sensitive-json-error", ErrInvalidResponse},
		{"wrong-role", 201, strings.Replace(string(valid), o.RoleNrn, "wrong", 1), ErrInvalidResponse},
		{"expired", 201, string(badExpiry), ErrInvalidResponse},
		{"oversized", 201, strings.Repeat("x", 16385), ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: original.CheckRedirect, Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": []string{"https://invalid.example"}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			})}
			got, err := CreateSession(context.Background(), o)
			if !errors.Is(err, tc.want) || calls != 1 {
				t.Fatalf("unexpected result: %v, calls %d", err, calls)
			}
			if err == nil && got != c {
				t.Fatal("credential mismatch")
			}
		})
	}
	client = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { return nil, errors.New("sensitive-network-error") })}
	if _, err := CreateSession(context.Background(), o); !errors.Is(err, ErrRequestFailed) || strings.Contains(err.Error(), "sensitive") {
		t.Fatal("raw network error leaked")
	}
}
