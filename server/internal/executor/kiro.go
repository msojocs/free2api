package executor

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/msojocs/free2api/server/internal/core"
	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/pkg/crypto"
	"github.com/msojocs/free2api/server/pkg/mailprovider"
	"golang.org/x/net/publicsuffix"
)

// Kiro (AWS Builder ID) registration.
// Reference: https://github.com/lxf746/any-auto-register/blob/main/platforms/kiro/core.py
const (
	kiroBase      = "https://app.kiro.dev"
	kiroSigninAWS = "https://us-east-1.signin.aws"
	kiroDirID     = "d-9067642ac7"
	kiroProfile   = "https://profile.aws.amazon.com"
	kiroUA        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// ---------------------------------------------------------------------------
// Minimal CBOR encoder (major types: map, text string, positive int)
// Supports only the subset needed for the Kiro InitiateLogin request.
// ---------------------------------------------------------------------------

// cborEncodeText encodes a UTF-8 text string (major type 3).
func cborEncodeText(s string) []byte {
	b := []byte(s)
	n := len(b)
	var head []byte
	switch {
	case n <= 23:
		head = []byte{0x60 | byte(n)}
	case n <= 0xFF:
		head = []byte{0x78, byte(n)}
	case n <= 0xFFFF:
		head = []byte{0x79, byte(n >> 8), byte(n)}
	default:
		head = []byte{0x7a, byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
	}
	return append(head, b...)
}

// cborEncodeMap encodes an ordered map of string→string pairs.
func cborEncodeMap(pairs [][2]string) []byte {
	n := len(pairs)
	var head []byte
	if n <= 23 {
		head = []byte{0xa0 | byte(n)}
	} else {
		head = []byte{0xb8, byte(n)}
	}
	var buf []byte
	buf = append(buf, head...)
	for _, p := range pairs {
		buf = append(buf, cborEncodeText(p[0])...)
		buf = append(buf, cborEncodeText(p[1])...)
	}
	return buf
}

// cborExtractString tries to find the value for a given key in a CBOR map
// by scanning the binary payload for the key's text encoding and reading
// the following text value. Fragile but sufficient for known responses.
func cborExtractString(data []byte, key string) string {
	needle := cborEncodeText(key)
	idx := bytes.Index(data, needle)
	if idx < 0 {
		return ""
	}
	// Skip past the key bytes, then read the following CBOR text item.
	pos := idx + len(needle)
	if pos >= len(data) {
		return ""
	}
	b := data[pos]
	major := b >> 5
	if major != 3 { // not a text string
		return ""
	}
	add := b & 0x1f
	var strLen int
	var bodyStart int
	switch {
	case add <= 23:
		strLen = int(add)
		bodyStart = pos + 1
	case add == 24:
		if pos+2 > len(data) {
			return ""
		}
		strLen = int(data[pos+1])
		bodyStart = pos + 2
	case add == 25:
		if pos+3 > len(data) {
			return ""
		}
		strLen = int(binary.BigEndian.Uint16(data[pos+1 : pos+3]))
		bodyStart = pos + 3
	default:
		return ""
	}
	if bodyStart+strLen > len(data) {
		return ""
	}
	return string(data[bodyStart : bodyStart+strLen])
}

// ---------------------------------------------------------------------------
// JWE RSA-OAEP-256 + A256GCM (using Go stdlib only)
// ---------------------------------------------------------------------------

// jweEncryptRSAOAEP256_A256GCM implements compact JWE serialization with
// RSA-OAEP-256 key wrapping and AES-256-GCM content encryption.
// publicKeyJWK is the JSON Web Key map from the Kiro API response.
func jweEncryptRSAOAEP256_A256GCM(plaintext []byte, publicKeyJWK map[string]interface{}) (string, error) {
	// Parse RSA public key from JWK
	rsaKey, kid, err := parseRSAPublicKeyFromJWK(publicKeyJWK)
	if err != nil {
		return "", fmt.Errorf("kiro JWE: parse key: %w", err)
	}

	// 1. Generate random 256-bit Content Encryption Key (CEK)
	cek := make([]byte, 32)
	if _, err := rand.Read(cek); err != nil {
		return "", err
	}

	// 2. Encrypt CEK with RSA-OAEP-256
	encryptedCEK, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaKey, cek, nil)
	if err != nil {
		return "", fmt.Errorf("kiro JWE: encrypt CEK: %w", err)
	}

	// 3. Build Protected Header and encode it
	protected := map[string]string{
		"alg": "RSA-OAEP-256",
		"kid": kid,
		"enc": "A256GCM",
		"cty": "enc",
		"typ": "application/aws+signin+jwe",
	}
	protectedJSON, _ := json.Marshal(protected)
	protectedB64 := base64.RawURLEncoding.EncodeToString(protectedJSON)

	// 4. Generate 96-bit IV
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	// 5. AES-256-GCM encrypt
	block, err := aes.NewCipher(cek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	// Additional data is the ASCII-encoded protected header
	aad := []byte(protectedB64)
	ciphertextAndTag := gcm.Seal(nil, iv, plaintext, aad)
	// GCM appends 16-byte auth tag at the end
	tagOffset := len(ciphertextAndTag) - gcm.Overhead()
	ciphertext := ciphertextAndTag[:tagOffset]
	authTag := ciphertextAndTag[tagOffset:]

	// 6. Compact serialization
	parts := []string{
		protectedB64,
		base64.RawURLEncoding.EncodeToString(encryptedCEK),
		base64.RawURLEncoding.EncodeToString(iv),
		base64.RawURLEncoding.EncodeToString(ciphertext),
		base64.RawURLEncoding.EncodeToString(authTag),
	}
	return strings.Join(parts, "."), nil
}

// parseRSAPublicKeyFromJWK parses an RSA public key from a JWK map.
// Supports "n"/"e" (JWK) and "x5c" (X.509 certificate chain) formats.
func parseRSAPublicKeyFromJWK(jwk map[string]interface{}) (*rsa.PublicKey, string, error) {
	kid, _ := jwk["kid"].(string)

	// Try x5c first (certificate chain)
	if x5c, ok := jwk["x5c"].([]interface{}); ok && len(x5c) > 0 {
		certDER, err := base64.StdEncoding.DecodeString(x5c[0].(string))
		if err != nil {
			return nil, kid, fmt.Errorf("decode x5c: %w", err)
		}
		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			return nil, kid, fmt.Errorf("parse cert: %w", err)
		}
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, kid, fmt.Errorf("x5c does not contain an RSA key")
		}
		return rsaKey, kid, nil
	}

	// Try PEM-encoded key
	if keyStr, ok := jwk["key"].(string); ok && keyStr != "" {
		block, _ := pem.Decode([]byte(keyStr))
		if block != nil {
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err == nil {
				if rsaKey, ok := pub.(*rsa.PublicKey); ok {
					return rsaKey, kid, nil
				}
			}
		}
	}

	// Try n/e (standard JWK)
	nStr, _ := jwk["n"].(string)
	eStr, _ := jwk["e"].(string)
	if nStr == "" || eStr == "" {
		return nil, kid, fmt.Errorf("JWK missing 'n' and/or 'e'")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, kid, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, kid, fmt.Errorf("decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	return &rsa.PublicKey{N: n, E: e}, kid, nil
}

// buildPasswordJWEPayload constructs the JWT-like plaintext that Kiro encrypts
// for password submission (matching the browser's PasswordEncryptor.encryptPassword).
func buildPasswordJWEPayload(password string) []byte {
	now := time.Now().Unix()
	jtiBytes := make([]byte, 16)
	_, _ = rand.Read(jtiBytes)
	jti := fmt.Sprintf("%x-%x-%x-%x-%x",
		jtiBytes[0:4], jtiBytes[4:6], jtiBytes[6:8], jtiBytes[8:10], jtiBytes[10:16])

	payload := map[string]interface{}{
		"iss":      "us-east-1.signin",
		"iat":      now,
		"nbf":      now,
		"jti":      jti,
		"exp":      now + 300,
		"aud":      "us-east-1.AWSPasswordService",
		"password": password,
	}
	b, _ := json.Marshal(payload)
	return b
}

// ---------------------------------------------------------------------------
// Kiro HTTP session
// ---------------------------------------------------------------------------

type kiroSession struct {
	noRedirect   *http.Client
	withRedirect *http.Client

	// State extracted during registration
	codeVerifier  string
	codeChallenge string
	pkceState     string
	visitorID     string

	wsh                  string // workflowStateHandle
	profileWfID          string // profile.aws.amazon.com workflow ID
	profileWfState       string
	portalCSRFToken      string
	orchestratorID       string
	workflowResultHandle string
	step11State          string
	awsUbidMain          string
	platformUbid         string
	awsD2CToken          string
	tesVisitorID         string
}

func newKiroSession(proxyURL string) (*kiroSession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("kiro: invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}
	noRedir := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	withRedir := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	// PKCE
	rawVerifier := make([]byte, 43)
	const pkceChrSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	for i := range rawVerifier {
		rawVerifier[i] = pkceChrSet[safeRandInt(len(pkceChrSet))]
	}
	verifier := string(rawVerifier)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	pkceState := fmt.Sprintf("%x", stateBytes)

	ts := time.Now().UnixMilli()
	vidSuffix := randAlphanumStr(12)
	visitorID := fmt.Sprintf("%d-%s", ts, vidSuffix)

	// platform-ubid
	platformUbid := fmt.Sprintf("%d-%d-%d",
		100+safeRandInt(900), 1000000+safeRandInt(9000000), 1000000+safeRandInt(9000000))

	return &kiroSession{
		noRedirect:    noRedir,
		withRedirect:  withRedir,
		codeVerifier:  verifier,
		codeChallenge: challenge,
		pkceState:     pkceState,
		visitorID:     visitorID,
		platformUbid:  platformUbid,
	}, nil
}

func (s *kiroSession) ua() map[string]string {
	return map[string]string{
		"User-Agent":       kiroUA,
		"Sec-CH-UA":        `"Chromium";v="131", "Not_A Brand";v="24"`,
		"Sec-CH-UA-Mobile": "?0",
	}
}

func (s *kiroSession) doReq(ctx context.Context, method, rawURL string, body io.Reader, headers map[string]string, followRedirects bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range s.ua() {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := s.noRedirect
	if followRedirects {
		client = s.withRedirect
	}
	return client.Do(req)
}

func (s *kiroSession) readJSON(resp *http.Response) (map[string]interface{}, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse JSON: %w (body: %s)", err, string(data[:min(200, len(data))]))
	}
	return result, nil
}

func (s *kiroSession) postJSON(ctx context.Context, rawURL string, payload interface{}, headers map[string]string, followRedirects bool) (map[string]interface{}, error) {
	b, _ := json.Marshal(payload)
	resp, err := s.doReq(ctx, http.MethodPost, rawURL, bytes.NewReader(b),
		mergeHeaders(headers, map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		}), followRedirects)
	if err != nil {
		return nil, err
	}
	return s.readJSON(resp)
}

func mergeHeaders(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// execAPI calls the signin.aws workflow execute endpoint.
func (s *kiroSession) execAPI(ctx context.Context, stepID string, inputs []interface{}, prefix, actionID string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/platform/%s%s/api/execute", kiroSigninAWS, kiroDirID, prefix)
	body := map[string]interface{}{
		"stepId":              stepID,
		"workflowStateHandle": s.wsh,
		"inputs":              inputs,
		"requestId":           newUUID(),
	}
	if actionID != "" {
		body["actionId"] = actionID
	}
	resp, err := s.doReq(ctx, http.MethodPost, apiURL,
		jsonBody(body),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"Origin":       kiroSigninAWS,
			"Referer":      fmt.Sprintf("%s/platform/%s/login", kiroSigninAWS, kiroDirID),
		}, true)
	if err != nil {
		return nil, err
	}
	d, err := s.readJSON(resp)
	if err != nil {
		return nil, err
	}
	if wsh, ok := d["workflowStateHandle"].(string); ok && wsh != "" {
		s.wsh = wsh
	}
	return d, nil
}

func jsonBody(v interface{}) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *kiroSession) fingerprintInput() map[string]interface{} {
	// Simplified fingerprint — real implementation uses XXTEA-encoded browser data.
	// This placeholder will be accepted by AWS but may trigger risk checks.
	return map[string]interface{}{
		"input_type":  "FingerPrintRequestInput",
		"fingerPrint": "",
	}
}

// step1KiroInit calls the Kiro InitiateLogin CBOR endpoint and returns the redirect URL.
func (s *kiroSession) step1KiroInit(ctx context.Context) (string, error) {
	body := cborEncodeMap([][2]string{
		{"idp", "BuilderId"},
		{"redirectUri", kiroBase + "/signin/oauth"},
		{"state", s.pkceState},
		{"codeChallenge", s.codeChallenge},
		{"codeChallengeMethod", "S256"},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		kiroBase+"/service/KiroWebPortalService/operation/InitiateLogin",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/cbor")
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("Smithy-Protocol", "rpc-v2-cbor")
	req.Header.Set("Origin", kiroBase)
	req.Header.Set("Referer", kiroBase+"/signin")
	req.Header.Set("User-Agent", kiroUA)
	req.Header.Set("X-Kiro-Visitorid", s.visitorID)
	req.AddCookie(&http.Cookie{Name: "kiro-visitor-id", Value: s.visitorID})

	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("kiro step1: %w", err)
	}
	defer resp.Body.Close()
	respData, _ := io.ReadAll(resp.Body)

	// Try CBOR extraction first, then JSON fallback.
	redirectURL := cborExtractString(respData, "redirectUrl")
	if redirectURL == "" {
		var j map[string]interface{}
		if err := json.Unmarshal(respData, &j); err == nil {
			redirectURL, _ = j["redirectUrl"].(string)
		}
	}
	if redirectURL == "" {
		return "", fmt.Errorf("kiro step1: redirectUrl not found in response (status=%d, body=%s)", resp.StatusCode, string(respData[:min(200, len(respData))]))
	}
	return redirectURL, nil
}

var wshRe = regexp.MustCompile(`workflowStateHandle=([^&#]+)`)
var wfIDRe = regexp.MustCompile(`workflowID=([^&#]+)`)

// step2GetWSH follows the redirect chain and extracts workflowStateHandle.
func (s *kiroSession) step2GetWSH(ctx context.Context, redirURL string) error {
	resp, err := s.doReq(ctx, http.MethodGet, redirURL, nil, nil, true)
	if err != nil {
		return fmt.Errorf("kiro step2: follow redir: %w", err)
	}
	defer resp.Body.Close()

	viewURL := resp.Request.URL.String()

	// Parse orchestrator_id from the view URL
	if u, err := url.Parse(viewURL); err == nil {
		qs := u.Query()
		if oid := qs.Get("orchestrator_id"); oid != "" {
			s.orchestratorID = oid
		}
	}

	// Build portal.sso URL
	vr := fmt.Sprintf("https://view.awsapps.com/start/#/?callback_url=&orchestrator_id=%s",
		url.QueryEscape(s.orchestratorID))
	portalURL := "https://portal.sso.us-east-1.amazonaws.com/login?directory_id=view&redirect_url=" +
		url.QueryEscape(vr)

	portalResp, err := s.doReq(ctx, http.MethodGet, portalURL, nil,
		map[string]string{
			"Accept": "*/*",
			"Origin": "https://view.awsapps.com",
		}, false)
	if err != nil {
		return fmt.Errorf("kiro step2 portal: %w", err)
	}
	defer portalResp.Body.Close()
	portalData, _ := io.ReadAll(portalResp.Body)

	var portalJSON map[string]interface{}
	if err := json.Unmarshal(portalData, &portalJSON); err == nil {
		if csrf, ok := portalJSON["csrfToken"].(string); ok {
			s.portalCSRFToken = csrf
		}
		if redir, ok := portalJSON["redirectUrl"].(string); ok {
			if m := wshRe.FindStringSubmatch(redir); len(m) >= 2 {
				s.wsh = m[1]
			}
			// Visit the redirect URL to get session cookies
			nextResp, err := s.doReq(ctx, http.MethodGet, redir, nil,
				map[string]string{"Accept": "text/html",
					"Referer": "https://portal.sso.us-east-1.amazonaws.com/"}, true)
			if err == nil {
				nextResp.Body.Close()
			}
		}
	}

	if s.wsh == "" {
		return fmt.Errorf("kiro step2: could not extract workflowStateHandle")
	}
	return nil
}

// step3SigninFlow runs the signin workflow to SIGNUP stage.
func (s *kiroSession) step3SigninFlow(ctx context.Context, email string) error {
	fpI := s.fingerprintInput()
	usrI := map[string]interface{}{
		"input_type": "UserRequestInput",
		"username":   email,
	}
	// 3a: init
	if _, err := s.execAPI(ctx, "", []interface{}{fpI}, "", ""); err != nil {
		return fmt.Errorf("kiro step3a: %w", err)
	}
	// 3b: start
	if _, err := s.execAPI(ctx, "start", []interface{}{fpI}, "", ""); err != nil {
		return fmt.Errorf("kiro step3b: %w", err)
	}
	// 3c: get-identity-user → SIGNUP
	d, err := s.execAPI(ctx, "get-identity-user", []interface{}{usrI, fpI}, "", "SIGNUP")
	if err != nil {
		return fmt.Errorf("kiro step3c: %w", err)
	}
	// Extract new WSH from redirect if present
	if redirMap, ok := d["redirect"].(map[string]interface{}); ok {
		if u, ok := redirMap["url"].(string); ok {
			if m := wshRe.FindStringSubmatch(u); len(m) >= 2 {
				s.wsh = m[1]
			}
		}
	}
	return nil
}

// step4SignupFlow runs the signup workflow and extracts the profile workflowID.
func (s *kiroSession) step4SignupFlow(ctx context.Context, email string) error {
	fpI := s.fingerprintInput()
	usrI := map[string]interface{}{
		"input_type": "UserRequestInput",
		"username":   email,
	}
	// 4a: init
	if _, err := s.execAPI(ctx, "", []interface{}{usrI, fpI}, "/signup", ""); err != nil {
		return fmt.Errorf("kiro step4a: %w", err)
	}
	// 4b: start
	d, err := s.execAPI(ctx, "start", []interface{}{usrI, fpI}, "/signup", "")
	if err != nil {
		return fmt.Errorf("kiro step4b: %w", err)
	}
	// Extract profile workflowID
	if redirMap, ok := d["redirect"].(map[string]interface{}); ok {
		if u, ok := redirMap["url"].(string); ok {
			if m := wfIDRe.FindStringSubmatch(u); len(m) >= 2 {
				s.profileWfID = m[1]
			}
		}
	}
	if s.profileWfID == "" {
		return fmt.Errorf("kiro step4: profile workflowID not found")
	}
	return nil
}

// step5GetTESToken obtains the TES visitor token from AWS.
func (s *kiroSession) step5GetTESToken(ctx context.Context) {
	resp, err := s.doReq(ctx, http.MethodPost,
		"https://vs.aws.amazon.com/token", jsonBody(map[string]string{}),
		map[string]string{
			"Content-Type": "application/json",
			"Origin":       kiroSigninAWS,
		}, true)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var j map[string]interface{}
	if err := json.Unmarshal(data, &j); err == nil {
		if token, ok := j["token"].(string); ok {
			s.awsD2CToken = token
			// Extract visitor ID from JWT
			parts := strings.Split(token, ".")
			if len(parts) >= 2 {
				padded := parts[1] + strings.Repeat("=", (4-len(parts[1])%4)%4)
				if decoded, err := base64.StdEncoding.DecodeString(padded); err == nil {
					var claims map[string]interface{}
					if err := json.Unmarshal(decoded, &claims); err == nil {
						if vid, ok := claims["vid"].(string); ok {
							s.tesVisitorID = vid
						}
					}
				}
			}
		}
	}
}

// profilePost sends a POST to profile.aws.amazon.com.
func (s *kiroSession) profilePost(ctx context.Context, path string, payload interface{}) (map[string]interface{}, error) {
	resp, err := s.doReq(ctx, http.MethodPost, kiroProfile+path, jsonBody(payload),
		map[string]string{
			"Content-Type": "application/json;charset=UTF-8",
			"Accept":       "*/*",
			"Origin":       kiroProfile,
			"Referer":      fmt.Sprintf("%s/?workflowID=%s", kiroProfile, s.profileWfID),
		}, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	_ = json.Unmarshal(data, &result)
	return result, nil
}

// step6ProfileLoad visits the profile page and calls /api/start.
func (s *kiroSession) step6ProfileLoad(ctx context.Context) error {
	if s.profileWfID == "" {
		return fmt.Errorf("kiro step6: no profileWfID")
	}

	// Load the profile page
	resp, err := s.doReq(ctx, http.MethodGet,
		fmt.Sprintf("%s?workflowID=%s", kiroProfile, s.profileWfID), nil,
		map[string]string{"Accept": "text/html"}, true)
	if err != nil {
		return fmt.Errorf("kiro step6 page: %w", err)
	}
	resp.Body.Close()

	time.Sleep(300 * time.Millisecond)

	// /api/start
	payload := map[string]interface{}{
		"workflowID":  s.profileWfID,
		"browserData": s.browserData("", "PageLoad"),
	}
	d, err := s.profilePost(ctx, "/api/start", payload)
	if err != nil {
		return fmt.Errorf("kiro step6 start: %w", err)
	}
	if wfState, ok := d["workflowState"].(string); ok {
		s.profileWfState = wfState
	}
	return nil
}

// browserData returns a minimal browser data payload for profile.aws.amazon.com.
func (s *kiroSession) browserData(pageName, eventType string) map[string]interface{} {
	attrs := map[string]interface{}{
		"fingerprint":     "",
		"eventTimestamp":  time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"timeSpentOnPage": "3000",
		"eventType":       eventType,
		"ubid":            s.platformUbid,
	}
	if pageName != "" {
		attrs["pageName"] = pageName
	}
	if s.tesVisitorID != "" {
		attrs["visitorId"] = s.tesVisitorID
	}
	return map[string]interface{}{"attributes": attrs, "cookies": map[string]string{}}
}

// step7SendOTP sends a verification code email via profile.aws.
func (s *kiroSession) step7SendOTP(ctx context.Context, email string) error {
	time.Sleep(2 * time.Second)
	payload := map[string]interface{}{
		"workflowState": s.profileWfState,
		"email":         email,
		"browserData":   s.browserData("EMAIL_COLLECTION", "PageSubmit"),
	}
	_, err := s.profilePost(ctx, "/api/send-otp", payload)
	return err
}

// step8CreateIdentity verifies the OTP and creates the identity.
func (s *kiroSession) step8CreateIdentity(ctx context.Context, otp, email, fullName string) (string, string, error) {
	time.Sleep(1 * time.Second)
	payload := map[string]interface{}{
		"workflowState": s.profileWfState,
		"userData":      map[string]string{"email": email, "fullName": fullName},
		"otpCode":       otp,
		"browserData":   s.browserData("EMAIL_VERIFICATION", "EmailVerification"),
	}
	d, err := s.profilePost(ctx, "/api/create-identity", payload)
	if err != nil {
		return "", "", fmt.Errorf("kiro step8: %w", err)
	}
	regCode, _ := d["registrationCode"].(string)
	signInState, _ := d["signInState"].(string)
	if regCode == "" || signInState == "" {
		return "", "", fmt.Errorf("kiro step8: missing registrationCode or signInState: %v", d)
	}
	return regCode, signInState, nil
}

// step9SignupRegistration exchanges the registrationCode for a password-setting handle.
func (s *kiroSession) step9SignupRegistration(ctx context.Context, regCode, signInState string) (map[string]interface{}, error) {
	// GET signup page
	signupURL := fmt.Sprintf("%s/platform/%s/signup?registrationCode=%s&state=%s",
		kiroSigninAWS, kiroDirID, url.QueryEscape(regCode), url.QueryEscape(signInState))
	getResp, err := s.doReq(ctx, http.MethodGet, signupURL, nil,
		map[string]string{"Accept": "text/html"}, true)
	if err != nil {
		return nil, fmt.Errorf("kiro step9 GET: %w", err)
	}
	getResp.Body.Close()

	time.Sleep(500 * time.Millisecond)

	fpI := s.fingerprintInput()
	regI := map[string]interface{}{
		"input_type":       "UserRegistrationRequestInput",
		"registrationCode": regCode,
		"state":            signInState,
	}

	apiURL := fmt.Sprintf("%s/platform/%s/signup/api/execute", kiroSigninAWS, kiroDirID)
	body := map[string]interface{}{
		"stepId":    "",
		"state":     signInState,
		"inputs":    []interface{}{regI, fpI},
		"requestId": newUUID(),
	}
	resp, err := s.doReq(ctx, http.MethodPost, apiURL, jsonBody(body),
		map[string]string{
			"Content-Type": "application/json; charset=UTF-8",
			"Accept":       "application/json, text/plain, */*",
			"Origin":       kiroSigninAWS,
			"Referer":      signupURL,
		}, true)
	if err != nil {
		return nil, fmt.Errorf("kiro step9 exec: %w", err)
	}
	d, err := s.readJSON(resp)
	if err != nil {
		return nil, fmt.Errorf("kiro step9 parse: %w", err)
	}
	if wsh, ok := d["workflowStateHandle"].(string); ok && wsh != "" {
		s.wsh = wsh
	}
	return d, nil
}

// step10SetPassword encrypts the password with JWE and submits it.
func (s *kiroSession) step10SetPassword(ctx context.Context, password, email string, step9Resp map[string]interface{}) (map[string]interface{}, error) {
	wsh, _ := step9Resp["workflowStateHandle"].(string)

	// Extract public key from encryptionContextResponse
	encCtxRaw, _ := step9Resp["workflowResponseData"].(map[string]interface{})
	encCtx, _ := encCtxRaw["encryptionContextResponse"].(map[string]interface{})
	pubKeyRaw, _ := encCtx["publicKey"].(map[string]interface{})
	if pubKeyRaw == nil {
		return nil, fmt.Errorf("kiro step10: no public key in response")
	}

	jwePayload := buildPasswordJWEPayload(password)
	jweToken, err := jweEncryptRSAOAEP256_A256GCM(jwePayload, pubKeyRaw)
	if err != nil {
		return nil, fmt.Errorf("kiro step10: JWE encrypt: %w", err)
	}

	fpI := s.fingerprintInput()
	pwdI := map[string]interface{}{
		"input_type":            "PasswordRequestInput",
		"password":              jweToken,
		"successfullyEncrypted": "SUCCESSFUL",
		"errorLog":              nil,
	}
	usrI := map[string]interface{}{
		"input_type": "UserRequestInput",
		"username":   email,
	}
	evtI := map[string]interface{}{
		"input_type":  "UserEventRequestInput",
		"directoryId": kiroDirID,
		"userName":    email,
		"userEvents": []map[string]interface{}{
			{
				"input_type":      "UserEvent",
				"eventType":       "PAGE_SUBMIT",
				"pageName":        "CREDENTIAL_COLLECTION",
				"timeSpentOnPage": 10000,
			},
		},
	}

	reqID := newUUID()
	apiURL := fmt.Sprintf("%s/platform/%s/signup/api/execute", kiroSigninAWS, kiroDirID)
	body := map[string]interface{}{
		"stepId":              "get-new-password-for-password-creation",
		"workflowStateHandle": wsh,
		"actionId":            "SUBMIT",
		"inputs":              []interface{}{pwdI, evtI, usrI, fpI},
		"visitorId":           s.tesVisitorID,
		"requestId":           reqID,
	}
	resp, err := s.doReq(ctx, http.MethodPost, apiURL, jsonBody(body),
		map[string]string{
			"Content-Type":     "application/json; charset=UTF-8",
			"Accept":           "application/json, text/plain, */*",
			"Origin":           kiroSigninAWS,
			"X-Amzn-RequestID": reqID,
		}, true)
	if err != nil {
		return nil, fmt.Errorf("kiro step10: %w", err)
	}
	d, err := s.readJSON(resp)
	if err != nil {
		return nil, fmt.Errorf("kiro step10 parse: %w", err)
	}

	// Extract workflowResultHandle
	if redirMap, ok := d["redirect"].(map[string]interface{}); ok {
		if u, ok := redirMap["url"].(string); ok {
			if parsed, err := url.Parse(u); err == nil {
				if wrh := parsed.Query().Get("workflowResultHandle"); wrh != "" {
					s.workflowResultHandle = wrh
				}
			}
		}
	}
	return d, nil
}

// step11FinalLogin completes the login workflow.
func (s *kiroSession) step11FinalLogin(ctx context.Context, email string, step10Resp map[string]interface{}) (map[string]interface{}, error) {
	redirMap, _ := step10Resp["redirect"].(map[string]interface{})
	if redirMap == nil {
		return nil, fmt.Errorf("kiro step11: no redirect in step10 response")
	}
	redirURL, _ := redirMap["url"].(string)
	parsed, err := url.Parse(redirURL)
	if err != nil {
		return nil, fmt.Errorf("kiro step11: parse redirect URL: %w", err)
	}
	qs := parsed.Query()
	loginWSH := qs.Get("workflowStateHandle")
	state := qs.Get("state")
	wfResult := qs.Get("workflowResultHandle")

	fpI := s.fingerprintInput()
	usrI := map[string]interface{}{
		"input_type": "UserRequestInput",
		"username":   email,
	}
	apiURL := fmt.Sprintf("%s/platform/%s/api/execute", kiroSigninAWS, kiroDirID)
	body := map[string]interface{}{
		"stepId":               "",
		"workflowStateHandle":  loginWSH,
		"workflowResultHandle": wfResult,
		"state":                state,
		"inputs":               []interface{}{usrI, fpI},
		"visitorId":            s.tesVisitorID,
		"requestId":            newUUID(),
	}
	resp, err := s.doReq(ctx, http.MethodPost, apiURL, jsonBody(body),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"Origin":       kiroSigninAWS,
		}, true)
	if err != nil {
		return nil, fmt.Errorf("kiro step11: %w", err)
	}
	d, err := s.readJSON(resp)
	if err != nil {
		return nil, fmt.Errorf("kiro step11 parse: %w", err)
	}

	// Capture final workflowResultHandle and state
	if redirMap, ok := d["redirect"].(map[string]interface{}); ok {
		if u, ok := redirMap["url"].(string); ok {
			if p, err := url.Parse(u); err == nil {
				if wrh := p.Query().Get("workflowResultHandle"); wrh != "" {
					s.workflowResultHandle = wrh
				}
				if st := p.Query().Get("state"); st != "" {
					s.step11State = st
				}
			}
		}
	}
	return d, nil
}

// step12GetTokens performs the OIDC auth code exchange to get Kiro access tokens.
func (s *kiroSession) step12GetTokens(ctx context.Context) (map[string]interface{}, error) {
	if s.portalCSRFToken == "" || s.workflowResultHandle == "" || s.step11State == "" {
		return nil, fmt.Errorf("kiro step12: missing state (csrf=%v, wrh=%v, state=%v)",
			s.portalCSRFToken != "", s.workflowResultHandle != "", s.step11State != "")
	}

	// 12a: POST portal.sso/auth/sso-token
	portalBase := "https://portal.sso.us-east-1.amazonaws.com"
	ssoBody := url.Values{
		"authCode": {s.workflowResultHandle},
		"state":    {s.step11State},
		"orgId":    {"view"},
	}
	resp, err := s.doReq(ctx, http.MethodPost,
		portalBase+"/auth/sso-token",
		strings.NewReader(ssoBody.Encode()),
		map[string]string{
			"Content-Type":         "application/x-www-form-urlencoded",
			"Accept":               "application/json, text/plain, */*",
			"X-Amz-SSO-CSRF-Token": s.portalCSRFToken,
			"Origin":               "https://view.awsapps.com",
			"Referer":              "https://view.awsapps.com/",
		}, true)
	if err != nil {
		return nil, fmt.Errorf("kiro step12a: %w", err)
	}
	ssoData, err := s.readJSON(resp)
	if err != nil {
		return nil, fmt.Errorf("kiro step12a parse: %w", err)
	}

	bearerToken, _ := ssoData["token"].(string)
	if bearerToken == "" {
		return nil, fmt.Errorf("kiro step12: no bearer token: %v", ssoData)
	}

	// 12b: whoAmI check
	whoResp, err := s.doReq(ctx, http.MethodGet,
		portalBase+"/token/whoAmI", nil,
		map[string]string{
			"Accept":        "application/json",
			"Authorization": "Bearer " + bearerToken,
		}, true)
	if err == nil {
		whoResp.Body.Close()
	}

	// 12c: POST oidc/authentication_result
	oidcBase := "https://oidc.us-east-1.amazonaws.com"
	authBody := map[string]interface{}{
		"bearerToken":    bearerToken,
		"orchestratorId": s.orchestratorID,
	}
	authResp, err := s.doReq(ctx, http.MethodPost,
		oidcBase+"/authentication_result", jsonBody(authBody),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "*/*",
		}, false)
	if err != nil {
		return nil, fmt.Errorf("kiro step12c: %w", err)
	}
	defer authResp.Body.Close()
	loc := authResp.Header.Get("Location")

	// 12d: Follow OIDC authorization redirect to get code
	if loc == "" {
		return nil, fmt.Errorf("kiro step12d: no location header from authentication_result")
	}
	codeResp, err := s.doReq(ctx, http.MethodGet, loc, nil, nil, false)
	if err != nil {
		return nil, fmt.Errorf("kiro step12d: %w", err)
	}
	defer codeResp.Body.Close()
	finalLoc := codeResp.Header.Get("Location")

	// Extract code from final redirect
	authCode := ""
	finalState := ""
	if finalLoc != "" {
		if p, err := url.Parse(finalLoc); err == nil {
			authCode = p.Query().Get("code")
			finalState = p.Query().Get("state")
		}
	}
	if authCode == "" {
		return nil, fmt.Errorf("kiro step12: no auth code in redirect: %s", finalLoc)
	}

	// 12e: POST ExchangeToken (CBOR)
	exchangeBody := cborEncodeMap([][2]string{
		{"code", authCode},
		{"codeVerifier", s.codeVerifier},
		{"state", finalState},
	})
	exchReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		kiroBase+"/service/KiroWebPortalService/operation/ExchangeToken",
		bytes.NewReader(exchangeBody))
	if err != nil {
		return nil, err
	}
	exchReq.Header.Set("Content-Type", "application/cbor")
	exchReq.Header.Set("Accept", "application/cbor")
	exchReq.Header.Set("Smithy-Protocol", "rpc-v2-cbor")
	exchReq.Header.Set("Origin", kiroBase)
	exchReq.Header.Set("User-Agent", kiroUA)

	exchResp, err := s.withRedirect.Do(exchReq)
	if err != nil {
		return nil, fmt.Errorf("kiro step12e: %w", err)
	}
	defer exchResp.Body.Close()
	exchData, _ := io.ReadAll(exchResp.Body)

	// Parse CBOR or JSON response
	accessToken := cborExtractString(exchData, "accessToken")
	sessionToken := cborExtractString(exchData, "sessionToken")
	csrfToken := cborExtractString(exchData, "csrfToken")

	if accessToken == "" {
		// Fallback: JSON
		var j map[string]interface{}
		if err := json.Unmarshal(exchData, &j); err == nil {
			accessToken, _ = j["accessToken"].(string)
			sessionToken, _ = j["sessionToken"].(string)
			csrfToken, _ = j["csrfToken"].(string)
		}
	}

	return map[string]interface{}{
		"accessToken":  accessToken,
		"sessionToken": sessionToken,
		"csrfToken":    csrfToken,
	}, nil
}

// ---------------------------------------------------------------------------
// KiroExecutor.Execute
// ---------------------------------------------------------------------------

// KiroExecutor registers new Kiro (AWS Builder ID) accounts.
type KiroExecutor struct{}

func NewKiroExecutor() *KiroExecutor {
	return &KiroExecutor{}
}

// Execute runs the full 12-step Kiro / AWS Builder ID registration flow.
//
// Config keys:
//
//	proxy, mail_provider, mail_api_url, mail_admin_token, mail_domain
//	kiro_name  – display name for the new account (default: "Kiro User")
//
// NOTE: This implementation omits the XXTEA device fingerprint (fwcim) that
// the reference Python implementation generates. As a result, AWS risk checks
// may trigger and reject the request. A residential proxy is recommended.
func (e *KiroExecutor) Execute(ctx context.Context, taskID uint, config map[string]interface{}, publish func(core.ProgressUpdate)) (*ExecutionResult, error) {
	sendProgress(publish, taskID, 0, "Starting Kiro (AWS Builder ID) account registration", "running")

	proxyURL := cfgStr(config, "proxy", "")
	fullName := cfgStr(config, "kiro_name", "Kiro User")

	// ── Temp email ────────────────────────────────────────────────────────────
	mailProviderType := cfgStr(config, "mail_provider", "mailtm")
	mailCfg := map[string]string{
		"api_url":     cfgStr(config, "mail_api_url", ""),
		"admin_token": cfgStr(config, "mail_admin_token", ""),
		"domain":      cfgStr(config, "mail_domain", ""),
	}
	if cfgBool(config, "mail_use_proxy", true) {
		mailCfg["proxy_url"] = proxyURL
	}
	mp, err := mailprovider.New(mailProviderType, mailCfg)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Mail provider error: %v", err), "failed")
		return nil, err
	}

	sendProgress(publish, taskID, 5, "Getting temporary email…", "running")
	mailAccount, err := mp.GetEmail(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Get email failed: %v", err), "failed")
		return nil, err
	}
	email := mailAccount.Email
	sendProgress(publish, taskID, 10, fmt.Sprintf("Got email: %s", email), "running")

	sess, err := newKiroSession(proxyURL)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Session init error: %v", err), "failed")
		return nil, err
	}

	// Step 1 – Kiro InitiateLogin (CBOR)
	sendProgress(publish, taskID, 12, "Step 1/12: Kiro InitiateLogin…", "running")
	redirectURL, err := sess.step1KiroInit(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 1 failed: %v", err), "failed")
		return nil, err
	}

	// Step 2 – follow redirect chain to get wsh
	sendProgress(publish, taskID, 16, "Step 2/12: Following redirect chain…", "running")
	if err := sess.step2GetWSH(ctx, redirectURL); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 2 failed: %v", err), "failed")
		return nil, err
	}

	// Step 3 – signin.aws SIGNUP workflow
	sendProgress(publish, taskID, 22, "Step 3/12: Signin workflow (SIGNUP)…", "running")
	if err := sess.step3SigninFlow(ctx, email); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 3 failed: %v", err), "failed")
		return nil, err
	}

	// Step 4 – signup workflow → get profile workflowID
	sendProgress(publish, taskID, 28, "Step 4/12: Signup workflow…", "running")
	if err := sess.step4SignupFlow(ctx, email); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 4 failed: %v", err), "failed")
		return nil, err
	}

	// Step 5 – TES token (optional)
	sendProgress(publish, taskID, 33, "Step 5/12: TES token…", "running")
	sess.step5GetTESToken(ctx)

	// Step 6 – profile.aws load + /api/start
	sendProgress(publish, taskID, 37, "Step 6/12: Profile page load…", "running")
	if err := sess.step6ProfileLoad(ctx); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 6 failed: %v", err), "failed")
		return nil, err
	}

	// Step 7 – send OTP
	sendProgress(publish, taskID, 43, "Step 7/12: Sending OTP email…", "running")
	if err := sess.step7SendOTP(ctx, email); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 7 failed: %v", err), "failed")
		return nil, err
	}

	// Wait for OTP
	sendProgress(publish, taskID, 48, "Waiting for OTP…", "running")
	otp, err := mp.WaitForCode(ctx, mailAccount, "", 120)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("OTP wait failed: %v", err), "failed")
		return nil, err
	}
	sendProgress(publish, taskID, 55, fmt.Sprintf("Got OTP: %s", otp), "running")

	// Step 8 – create identity
	sendProgress(publish, taskID, 58, "Step 8/12: Creating identity…", "running")
	regCode, signInState, err := sess.step8CreateIdentity(ctx, otp, email, fullName)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 8 failed: %v", err), "failed")
		return nil, err
	}

	// Step 9 – signup registration
	sendProgress(publish, taskID, 63, "Step 9/12: Signup registration…", "running")
	step9Resp, err := sess.step9SignupRegistration(ctx, regCode, signInState)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 9 failed: %v", err), "failed")
		return nil, err
	}

	// Step 10 – set password (JWE encrypted)
	password := randPassword()
	sendProgress(publish, taskID, 70, "Step 10/12: Setting password (JWE)…", "running")
	step10Resp, err := sess.step10SetPassword(ctx, password, email, step9Resp)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 10 failed: %v", err), "failed")
		return nil, err
	}

	// Step 11 – final login
	sendProgress(publish, taskID, 80, "Step 11/12: Final login…", "running")
	if _, err := sess.step11FinalLogin(ctx, email, step10Resp); err != nil {
		// Non-fatal: account may already be created
		sendProgress(publish, taskID, 82, fmt.Sprintf("Step 11 note: %v", err), "running")
	}

	// Step 12 – get tokens
	sendProgress(publish, taskID, 85, "Step 12/12: Exchanging tokens…", "running")
	tokens, tokenErr := sess.step12GetTokens(ctx)
	if tokenErr != nil {
		sendProgress(publish, taskID, 88, fmt.Sprintf("Token exchange note: %v", tokenErr), "running")
		tokens = map[string]interface{}{}
	}

	// Persist
	encPass, err := crypto.Encrypt(password)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Encrypt error: %v", err), "failed")
		return nil, err
	}
	extraMap := map[string]interface{}{
		"name":         fullName,
		"accessToken":  tokens["accessToken"],
		"sessionToken": tokens["sessionToken"],
		"csrfToken":    tokens["csrfToken"],
	}
	extraJSON, _ := json.Marshal(extraMap)
	acct := &model.Account{
		Email:       email,
		Password:    encPass,
		Type:        "kiro",
		Status:      "active",
		TaskBatchID: taskID,
		Extra:       string(extraJSON),
	}

	accessToken, _ := tokens["accessToken"].(string)
	msg := fmt.Sprintf("✓ Kiro account registered: %s", email)
	if accessToken != "" {
		msg += fmt.Sprintf(" (token=%s…)", accessToken[:min(20, len(accessToken))])
	}

	return &ExecutionResult{
		Account:        acct,
		SuccessMessage: msg,
	}, nil
}
