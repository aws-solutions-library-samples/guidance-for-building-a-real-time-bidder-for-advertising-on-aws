package lowlevel

import (
	"bytes"
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	dynamodbV1 "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"github.com/valyala/fasttemplate"
)

// LowLevel is the class responsible for performing low level DynamoDB API queries.
// This class methods are NOT safe for concurrent calls.
type LowLevel struct {
	awsConfig   aws.Config
	url         string
	host        string
	credentials aws.Credentials

	httpClient     *fasthttp.Client
	signer         *Signer
	bodyBuilder    *fasttemplate.Template
	responseParser *fastjson.Parser

	// Reusable buffers used to avoid allocation.
	templateBuffer *bytes.Buffer
	base64Buffer   []byte
}

// New initializes LowLevel instance.
func New(cfg Config, tableName string, awsConfig aws.Config) (*LowLevel, error) {
	url, err := resolveEndpoint(awsConfig)
	if err != nil {
		return nil, err
	}

	credentials, err := awsConfig.Credentials.Retrieve(context.Background())
	if err != nil {
		return nil, err
	}

	urlParts := strings.SplitAfter(url, "://")
	host := urlParts[len(urlParts)-1]

	return &LowLevel{
		awsConfig:   awsConfig,
		url:         url,
		host:        host,
		credentials: credentials,

		httpClient: &fasthttp.Client{
			MaxConnsPerHost: cfg.MaxConnsPerHost,
			ReadTimeout:     cfg.ReadWriteTimeout,
			WriteTimeout:    cfg.ReadWriteTimeout,
		},
		signer:         NewSinger(awsConfig.Region, host, credentials),
		bodyBuilder:    getBodyTemplate(tableName),
		responseParser: &fastjson.Parser{},

		templateBuffer: &bytes.Buffer{},
	}, nil
}

// resolveEndpoint uses EndpointResolver from aws.Config to resolve dynamodb endpoint.
func resolveEndpoint(awsConfig aws.Config) (string, error) {
	if awsConfig.EndpointResolver == nil {
		return resolveEndpointFallback(awsConfig)
	}

	endpoint, err := awsConfig.EndpointResolver.
		ResolveEndpoint(dynamodb.ServiceID, awsConfig.Region)
	if err != nil {
		return resolveEndpointFallback(awsConfig)
	}

	return endpoint.URL, nil
}

// resolveEndpointFallback is a fallback to default endpoint resolver.
func resolveEndpointFallback(awsConfig aws.Config) (string, error) {
	endpoint, err := endpoints.DefaultResolver().
		EndpointFor(dynamodbV1.ServiceName, awsConfig.Region)
	if err != nil {
		return "", err
	}
	return endpoint.URL, nil
}
