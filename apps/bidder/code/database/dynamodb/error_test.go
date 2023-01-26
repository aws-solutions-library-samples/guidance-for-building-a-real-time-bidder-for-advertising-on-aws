package dynamodb

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/stretchr/testify/assert"
)

// Test if error returned by DAX in case of request timeout is decoded properly.
func TestDAXTimeout(t *testing.T) {
	err := awserr.New(request.CanceledErrorCode, "request context canceled", context.DeadlineExceeded)
	awsErr, ok := err.(awserr.Error)
	assert.True(t, ok && awsErr.OrigErr() == context.DeadlineExceeded)
}
