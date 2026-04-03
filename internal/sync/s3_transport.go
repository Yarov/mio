package sync

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// S3Transport implements Transport using an S3-compatible object store.
// It uses AWS Signature V4 for authentication with no external dependencies.
type S3Transport struct {
	endpoint  string
	bucket    string
	accessKey string
	secretKey string
	region    string
	client    *http.Client
}

// NewS3Transport creates a new S3Transport.
func NewS3Transport(endpoint, bucket, accessKey, secretKey, region string) (*S3Transport, error) {
	if bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}
	if endpoint == "" {
		// Default to AWS S3
		endpoint = fmt.Sprintf("https://s3.%s.amazonaws.com", region)
	}
	if region == "" {
		region = "us-east-1"
	}
	// Strip trailing slash
	endpoint = strings.TrimRight(endpoint, "/")

	return &S3Transport{
		endpoint:  endpoint,
		bucket:    bucket,
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		client:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *S3Transport) objectKey(chunkID string) string {
	return "chunks/" + chunkID + ".jsonl.gz"
}

func (t *S3Transport) objectURL(key string) string {
	return fmt.Sprintf("%s/%s/%s", t.endpoint, t.bucket, key)
}

func (t *S3Transport) Write(chunkID string, data []byte) error {
	// Compress the data
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return fmt.Errorf("compress data: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}

	body := buf.Bytes()
	key := t.objectKey(chunkID)
	url := t.objectURL(key)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	t.signRequest(req, body)

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("PUT object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT object %s: HTTP %d: %s", key, resp.StatusCode, string(respBody))
	}

	return nil
}

func (t *S3Transport) Read(chunkID string) ([]byte, error) {
	key := t.objectKey(chunkID)
	url := t.objectURL(key)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	t.signRequest(req, nil)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET object %s: HTTP %d: %s", key, resp.StatusCode, string(respBody))
	}

	compressed, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

// listBucketResult is the XML response from S3 ListObjectsV2.
type listBucketResult struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	Contents              []s3Object     `xml:"Contents"`
	IsTruncated           bool           `xml:"IsTruncated"`
	NextContinuationToken string         `xml:"NextContinuationToken"`
}

type s3Object struct {
	Key string `xml:"Key"`
}

func (t *S3Transport) List() ([]string, error) {
	var allIDs []string
	continuationToken := ""

	for {
		url := fmt.Sprintf("%s/%s?list-type=2&prefix=chunks/", t.endpoint, t.bucket)
		if continuationToken != "" {
			url += "&continuation-token=" + continuationToken
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		t.signRequest(req, nil)

		resp, err := t.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("list objects: HTTP %d: %s", resp.StatusCode, string(body))
		}
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		var result listBucketResult
		if err := xml.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse list response: %w", err)
		}

		for _, obj := range result.Contents {
			key := obj.Key
			if strings.HasPrefix(key, "chunks/") && strings.HasSuffix(key, ".jsonl.gz") {
				name := strings.TrimPrefix(key, "chunks/")
				id := name[:len(name)-len(".jsonl.gz")]
				allIDs = append(allIDs, id)
			}
		}

		if !result.IsTruncated {
			break
		}
		continuationToken = result.NextContinuationToken
	}

	return allIDs, nil
}

func (t *S3Transport) Exists(chunkID string) (bool, error) {
	key := t.objectKey(chunkID)
	url := t.objectURL(key)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	t.signRequest(req, nil)

	resp, err := t.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("HEAD object: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}
	return false, fmt.Errorf("HEAD object %s: HTTP %d", key, resp.StatusCode)
}

// --- AWS Signature V4 implementation ---

func (t *S3Transport) signRequest(req *http.Request, payload []byte) {
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	req.Header.Set("x-amz-date", amzdate)
	req.Header.Set("Host", req.URL.Host)

	// Payload hash
	var payloadHash string
	if payload != nil {
		h := sha256.Sum256(payload)
		payloadHash = hex.EncodeToString(h[:])
	} else {
		h := sha256.Sum256([]byte(""))
		payloadHash = hex.EncodeToString(h[:])
	}
	req.Header.Set("x-amz-content-sha256", payloadHash)

	// Canonical request
	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQueryString := req.URL.RawQuery

	// Signed headers
	signedHeaderKeys := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	if req.Header.Get("Content-Type") != "" {
		signedHeaderKeys = append(signedHeaderKeys, "content-type")
	}
	sort.Strings(signedHeaderKeys)
	signedHeaders := strings.Join(signedHeaderKeys, ";")

	var canonicalHeaders strings.Builder
	for _, k := range signedHeaderKeys {
		var v string
		if k == "host" {
			v = req.URL.Host
		} else {
			v = req.Header.Get(http.CanonicalHeaderKey(k))
		}
		canonicalHeaders.WriteString(k + ":" + strings.TrimSpace(v) + "\n")
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to sign
	credentialScope := datestamp + "/" + t.region + "/s3/aws4_request"
	canonHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		hex.EncodeToString(canonHash[:]),
	}, "\n")

	// Signing key
	signingKey := t.deriveSigningKey(datestamp)

	// Signature
	sig := hmacSHA256(signingKey, []byte(stringToSign))
	signature := hex.EncodeToString(sig)

	// Authorization header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		t.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func (t *S3Transport) deriveSigningKey(datestamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+t.secretKey), []byte(datestamp))
	kRegion := hmacSHA256(kDate, []byte(t.region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

