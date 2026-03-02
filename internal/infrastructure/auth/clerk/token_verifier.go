package clerk

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const jwksCacheTTL = 15 * time.Minute

type TokenClaims struct {
	Subject string
	Issuer  string
	Expires time.Time
}

type TokenVerifier struct {
	jwksURL string
	issuer  string
	client  *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func NewTokenVerifier(jwksURL, issuer string) *TokenVerifier {
	return &TokenVerifier{
		jwksURL: jwksURL,
		issuer:  issuer,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		keys: make(map[string]*rsa.PublicKey),
	}
}

func (v *TokenVerifier) Verify(ctx context.Context, token string) (*TokenClaims, error) {
	header, payload, signature, signingInput, err := parseJWT(token)
	if err != nil {
		return nil, err
	}
	if header.Alg != "RS256" {
		return nil, errors.New("unsupported JWT alg")
	}
	if header.KID == "" {
		return nil, errors.New("missing JWT key id")
	}
	if payload.Sub == "" {
		return nil, errors.New("missing JWT subject")
	}
	if payload.Exp <= 0 {
		return nil, errors.New("missing JWT exp claim")
	}
	if v.issuer != "" && payload.Iss != v.issuer {
		return nil, errors.New("invalid JWT issuer")
	}

	now := time.Now().Unix()
	if payload.Exp <= now {
		return nil, errors.New("token expired")
	}
	if payload.Nbf > 0 && payload.Nbf > now {
		return nil, errors.New("token not active yet")
	}

	key, err := v.keyByID(ctx, header.KID)
	if err != nil {
		return nil, err
	}

	hashed := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], signature); err != nil {
		return nil, errors.New("invalid token signature")
	}

	return &TokenClaims{
		Subject: payload.Sub,
		Issuer:  payload.Iss,
		Expires: time.Unix(payload.Exp, 0),
	}, nil
}

func (v *TokenVerifier) keyByID(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	cacheValid := time.Now().Before(v.expiresAt)
	v.mu.RUnlock()
	if ok && cacheValid {
		return key, nil
	}

	if err := v.refreshJWKS(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	if !ok {
		return nil, errors.New("signing key not found")
	}
	return key, nil
}

func (v *TokenVerifier) refreshJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: status %d", resp.StatusCode)
	}

	type jwk struct {
		Kty string `json:"kty"`
		KID string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	type jwksResponse struct {
		Keys []jwk `json:"keys"`
	}

	var parsed jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, candidate := range parsed.Keys {
		if candidate.Kty != "RSA" || candidate.KID == "" || candidate.N == "" || candidate.E == "" {
			continue
		}

		pub, err := rsaPublicKeyFromJWK(candidate.N, candidate.E)
		if err != nil {
			continue
		}
		keys[candidate.KID] = pub
	}

	if len(keys) == 0 {
		return errors.New("JWKS did not contain valid RSA keys")
	}

	v.mu.Lock()
	v.keys = keys
	v.expiresAt = time.Now().Add(jwksCacheTTL)
	v.mu.Unlock()

	return nil
}

func rsaPublicKeyFromJWK(nEncoded, eEncoded string) (*rsa.PublicKey, error) {
	nRaw, err := base64.RawURLEncoding.DecodeString(nEncoded)
	if err != nil {
		return nil, fmt.Errorf("decode JWK modulus: %w", err)
	}
	eRaw, err := base64.RawURLEncoding.DecodeString(eEncoded)
	if err != nil {
		return nil, fmt.Errorf("decode JWK exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nRaw)
	e := new(big.Int).SetBytes(eRaw)
	if n.Sign() <= 0 || e.Sign() <= 0 {
		return nil, errors.New("invalid JWK RSA values")
	}
	if !e.IsInt64() {
		return nil, errors.New("JWK exponent too large")
	}

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	KID string `json:"kid"`
}

type jwtPayload struct {
	Sub string `json:"sub"`
	Iss string `json:"iss"`
	Exp int64  `json:"exp"`
	Nbf int64  `json:"nbf"`
}

func parseJWT(token string) (jwtHeader, jwtPayload, []byte, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtHeader{}, jwtPayload{}, nil, "", errors.New("invalid JWT format")
	}

	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, jwtPayload{}, nil, "", errors.New("decode JWT header")
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtHeader{}, jwtPayload{}, nil, "", errors.New("decode JWT payload")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, jwtPayload{}, nil, "", errors.New("decode JWT signature")
	}

	var header jwtHeader
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return jwtHeader{}, jwtPayload{}, nil, "", errors.New("parse JWT header")
	}
	var payload jwtPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return jwtHeader{}, jwtPayload{}, nil, "", errors.New("parse JWT payload")
	}

	return header, payload, signature, parts[0] + "." + parts[1], nil
}
