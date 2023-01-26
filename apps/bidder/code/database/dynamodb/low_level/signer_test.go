package lowlevel

import (
	"bytes"
	"context"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// Test if authorization header produced by Signer
// is equal to one produced by AWS SDK Signer.
func TestSignRequest(t *testing.T) {
	credentials := aws.Credentials{
		AccessKeyID:     "test-access-key-id",
		SecretAccessKey: "test-secret-access-key",
		SessionToken:    "test-session-token",
	}

	body := []byte("test-body")
	region := "us-east-1"
	host := "dynamodb.us-east-1.amazonaws.com"

	signingTime := time.Now()

	signer := NewSinger(region, host, credentials)
	actualAuthHeader, actualDateHeader, err := signRequest(signer, body, signingTime)
	assert.NoError(t, err)

	expectedAuthHeader, expectedDateHeader, err := signRequestSDK(
		body, signingTime, credentials, region, "https://"+host)
	assert.NoError(t, err)

	assert.Equal(t, expectedAuthHeader, actualAuthHeader)
	assert.Equal(t, expectedDateHeader, actualDateHeader)

	// Run comparison again, to check if signer
	// isn't left in a broken state after first run.
	actualAuthHeader, actualDateHeader, err = signRequest(signer, body, signingTime)
	assert.NoError(t, err)

	assert.Equal(t, expectedAuthHeader, actualAuthHeader)
	assert.Equal(t, expectedDateHeader, actualDateHeader)
}

// signRequest is a wrapper around low-level Signer allowing
// to get correct authorization headers for bidder-specific queries.
func signRequest(signer *Signer, body []byte, signingTime time.Time,
) (authHeader, signingTimeHeader string, err error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	if err := signer.SignRequest(body, signingTime, req); err != nil {
		return "", "", err
	}

	return string(req.Header.Peek("Authorization")),
		string(req.Header.Peek("X-Amz-Date")), nil
}

// signRequestSDK is a wrapper around AWS SDK Signer allowing
// to get correct authorization headers for bidder-specific queries.
func signRequestSDK(
	body []byte,
	signingTime time.Time,
	credentials aws.Credentials,
	region string,
	url string,
) (authHeader, signingTimeHeader string, err error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}

	payloadHash := newHasher().sha256(body, new([]byte))

	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("Accept-Encoding", "identity")

	signer := v4.NewSigner()

	err = signer.SignHTTP(
		context.Background(),
		credentials,
		req,
		hex.EncodeToString(payloadHash),
		"dynamodb",
		region,
		signingTime,
	)
	if err != nil {
		return "", "", err
	}

	return req.Header.Get("Authorization"), req.Header.Get("X-Amz-Date"), nil
}
