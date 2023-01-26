package lowlevel

import (
	"bidder/code/database/api"
	"bidder/code/id"
	"context"
	"encoding/base64"
	"io"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/valyala/fasthttp"
)

// GetDevice performs DynamoDB low-level API device query.
// Low-level query does not take deadline as an argument.
// Timeout is defined on http client level, which results
// in a coarser timeout enforcing, but improves performance.
func (ll *LowLevel) GetDevice(deviceID id.ID, result *[]id.ID) error {
	err := ll.getDevice(deviceID, result)
	if err != nil {
		return errors.Wrap(err, "error while querying dynamodb low-level api")
	}
	return nil
}

func (ll *LowLevel) getDevice(deviceID id.ID, result *[]id.ID) error {
	if err := ll.renewCredentials(); err != nil {
		return err
	}

	body, err := ll.buildBody(deviceID)
	if err != nil {
		return err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(ll.url)
	req.SetBody(body)
	req.Header.SetMethodBytes([]byte("POST"))
	req.Header.SetContentLength(len(body))
	req.Header.SetHost(ll.host)
	req.Header.Set(amzTargetKey, "DynamoDB_20120810.GetItem")
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("Accept-Encoding", "identity")

	if err := ll.signer.SignRequest(body, time.Now(), req); err != nil {
		return err
	}

	if err := ll.httpClient.Do(req, resp); err != nil {
		if errors.Is(err, fasthttp.ErrTimeout) ||
			errors.Is(err, fasthttp.ErrTLSHandshakeTimeout) ||
			errors.Is(err, fasthttp.ErrDialTimeout) ||
			strings.Contains(err.Error(), "i/o timeout") {
			return api.ErrTimeout
		}

		return err
	}

	return ll.parseResponse(resp.Body(), result)
}

func (ll *LowLevel) buildBody(deviceID id.ID) ([]byte, error) {
	ll.templateBuffer.Reset()
	_, err := ll.bodyBuilder.ExecuteFunc(
		ll.templateBuffer,
		func(w io.Writer, tag string) (int, error) {
			if tag == tagID {
				resizeBuffer(&ll.base64Buffer, base64.StdEncoding.EncodedLen(len(deviceID)))
				base64.StdEncoding.Encode(ll.base64Buffer, deviceID[:])
				return w.Write(ll.base64Buffer)
			}
			return 0, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return ll.templateBuffer.Bytes(), nil
}

func (ll *LowLevel) parseResponse(body []byte, result *[]id.ID) error {
	v, err := ll.responseParser.ParseBytes(body)
	if err != nil {
		return err
	}

	// Check for error.
	errorMessage := v.GetStringBytes("message")
	if errorMessage != nil {
		return errors.New(string(errorMessage))
	}

	audiences := v.GetStringBytes("Item", "a", "B")
	if audiences == nil {
		return api.ErrItemNotFound
	}

	resizeBuffer(&ll.base64Buffer, base64.StdEncoding.DecodedLen(len(audiences)))
	if _, err := base64.StdEncoding.Decode(ll.base64Buffer, audiences); err != nil {
		return err
	}

	ID := id.ID{}
	for offset := 0; offset < len(ll.base64Buffer); offset += id.Len {
		copy(ID[:], ll.base64Buffer[offset:])
		*result = append(*result, ID)
	}

	return nil
}

func (ll *LowLevel) renewCredentials() error {
	if ll.credentials.Expired() {
		var err error
		ll.credentials, err = ll.awsConfig.Credentials.Retrieve(context.Background())
		if err != nil {
			return errors.Wrap(err, "error while retrieving credentials")
		}
		// Signers credentials can't be changed, so a new Signer is required.
		ll.signer = NewSinger(ll.awsConfig.Region, ll.host, ll.credentials)
	}

	return nil
}
