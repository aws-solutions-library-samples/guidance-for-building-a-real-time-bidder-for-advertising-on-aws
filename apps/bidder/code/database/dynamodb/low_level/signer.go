package lowlevel

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasttemplate"
)

//nolint:gosec // no, these are not hardcoded credentials
const (
	longTimeFormat  = "20060102T150405Z"
	shortTimeFormat = "20060102"

	amzDateKey          = "x-amz-date"
	amzTargetKey        = "x-amz-target"
	amzSecurityTokenKey = "x-amz-security-token"
	authKey             = "authorization"
)

// Signer creates AWS v4 signing headers for DynamoDB GetItem request.
// In contrast to AWS SDK Signer, it works only for bidder-specific
// requests. This allows to hardcode parts of the request data and
// assume some (eg. region, credentials) do not change between signer runs.
type Signer struct {
	region      string
	credentials aws.Credentials

	hasher  *hasher
	key     []byte
	keyDate []byte

	canonicalStringBuilder *fasttemplate.Template
	strToSignBuilder       *fasttemplate.Template
	authHeaderBuilder      *fasttemplate.Template

	// Reusable buffers used to avoid allocation.
	hashBuffer      []byte
	hexEncodeBuffer []byte
	templateBuffer  *bytes.Buffer

	// Variable-specific buffers used to avoid allocation.
	timeLong  []byte
	timeShort []byte
}

// NewSinger creates new Signer.
func NewSinger(region, host string, credentials aws.Credentials) *Signer {
	return &Signer{
		region:      region,
		credentials: credentials,

		hasher: newHasher(),

		canonicalStringBuilder: getCanonicalStringTemplate(host, credentials),
		strToSignBuilder:       getStrToSignTemplate(region),
		authHeaderBuilder:      getAuthHeaderTemplate(region, credentials),

		templateBuffer: &bytes.Buffer{},
	}
}

// SignRequest creates AWS v4 signing headers for DynamoDB GetItem request.
func (s *Signer) SignRequest(body []byte, signingTime time.Time, req *fasthttp.Request,
) error {
	resizeBuffer(&s.timeLong, 0)
	resizeBuffer(&s.timeShort, 0)

	s.timeLong = signingTime.UTC().AppendFormat(s.timeLong, longTimeFormat)
	s.timeShort = signingTime.UTC().AppendFormat(s.timeShort, shortTimeFormat)

	bodyHash := s.hasher.sha256(body, &s.hashBuffer)
	bodyHashHex := hexToBuffer(bodyHash, &s.hexEncodeBuffer)

	canonicalString, err := s.buildCanonicalString(bodyHashHex, s.timeLong, len(body))
	if err != nil {
		return err
	}

	canonicalStringHash := s.hasher.sha256(canonicalString, &s.hashBuffer)
	canonicalStringHashHex := hexToBuffer(canonicalStringHash, &s.hexEncodeBuffer)

	strToSign, err := s.buildStringToSign(canonicalStringHashHex, s.timeLong, s.timeShort)
	if err != nil {
		return err
	}

	signingSignature := s.buildSignature(strToSign, s.timeShort)
	authHeader, err := s.buildAuthHeader(signingSignature, s.timeShort)
	if err != nil {
		return err
	}

	// Set authorization headers.
	req.Header.SetBytesV(amzDateKey, s.timeLong)
	req.Header.SetBytesV(authKey, authHeader)
	if s.credentials.SessionToken != "" {
		req.Header.Set(amzSecurityTokenKey, s.credentials.SessionToken)
	}

	return nil
}

func (s *Signer) buildCanonicalString(payloadHash, timeLong []byte, contentLen int,
) ([]byte, error) {
	s.templateBuffer.Reset()
	_, err := s.canonicalStringBuilder.ExecuteFunc(
		s.templateBuffer,
		func(w io.Writer, tag string) (int, error) {
			switch tag {
			case tagLength:
				return fmt.Fprint(w, contentLen)
			case tagTimeLong:
				return w.Write(timeLong)
			case tagPayloadHash:
				return w.Write(payloadHash)
			}
			return 0, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return s.templateBuffer.Bytes(), nil
}

func (s *Signer) buildStringToSign(canonicalStringHash, timeLong, timeShort []byte,
) ([]byte, error) {
	s.templateBuffer.Reset()
	_, err := s.strToSignBuilder.ExecuteFunc(s.templateBuffer,
		func(w io.Writer, tag string) (int, error) {
			switch tag {
			case tagTimeLong:
				return w.Write(timeLong)
			case tagTimeShort:
				return w.Write(timeShort)
			case tagCanonicalStringHash:
				return w.Write(canonicalStringHash)
			}
			return 0, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return s.templateBuffer.Bytes(), nil
}

func (s *Signer) buildSignature(strToSign, timeShort []byte) []byte {
	if !bytes.Equal(timeShort, s.keyDate) {
		// Key contains encrypted short time, so it needs to be recalculated each time day changes.
		hmacDate := s.hasher.hmacsha256([]byte("AWS4"+s.credentials.SecretAccessKey), timeShort, &s.hashBuffer)
		hmacRegion := s.hasher.hmacsha256(hmacDate, []byte(s.region), &s.hashBuffer)
		hmacService := s.hasher.hmacsha256(hmacRegion, []byte(dynamodb.ServiceName), &s.hashBuffer)
		key := s.hasher.hmacsha256(hmacService, []byte("aws4_request"), &s.hashBuffer)

		s.key = make([]byte, len(key))
		copy(s.key, key)

		s.keyDate = make([]byte, len(timeShort))
		copy(s.keyDate, timeShort)
	}

	authHash := s.hasher.hmacsha256(s.key, strToSign, &s.hashBuffer)
	return hexToBuffer(authHash, &s.hexEncodeBuffer)
}

func (s *Signer) buildAuthHeader(signature, timeShort []byte) ([]byte, error) {
	s.templateBuffer.Reset()
	_, err := s.authHeaderBuilder.ExecuteFunc(s.templateBuffer,
		func(w io.Writer, tag string) (int, error) {
			switch tag {
			case tagTimeShort:
				return w.Write(timeShort)
			case tagSignature:
				return w.Write(signature)
			}
			return 0, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return s.templateBuffer.Bytes(), nil
}

func hexToBuffer(data []byte, buffer *[]byte) []byte {
	resizeBuffer(buffer, hex.EncodedLen(len(data)))
	hex.Encode(*buffer, data)
	return *buffer
}

func resizeBuffer(buffer *[]byte, length int) {
	if cap(*buffer) < length {
		*buffer = make([]byte, length)
	}
	*buffer = (*buffer)[:length]
}
