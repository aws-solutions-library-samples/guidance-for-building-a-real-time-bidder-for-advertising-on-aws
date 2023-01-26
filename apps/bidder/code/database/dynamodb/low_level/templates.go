package lowlevel

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/valyala/fasttemplate"
)

const (
	startTag = "["
	endTag   = "]"
)

// constant data tags
const (
	tagTableName                    = "table-name"
	tagHost                         = "host"
	tagRegion                       = "region"
	tagSecurityTokenHeader          = "security-token-header"
	tagSecurityTokenCanonicalHeader = "security-token-canonical-header"
)

const (
	tagID                  = "id"
	tagTimeShort           = "time-short"
	tagTimeLong            = "time-long"
	tagCanonicalStringHash = "canonical-string-hash"
	tagLength              = "length"
	tagPayloadHash         = "payload-hash"
	tagAccessKeyID         = "access-key-id"
	tagSignature           = "signature"
)

func getBodyTemplate(tableName string) *fasttemplate.Template {
	const bodyTemplate = `{"Key":{"d":{"B":"[id]"}},"TableName":"[table-name]"}`

	precompiled := fasttemplate.ExecuteStringStd(bodyTemplate, startTag, endTag,
		map[string]interface{}{
			tagTableName: tableName,
		})

	return fasttemplate.New(precompiled, startTag, endTag)
}

func getStrToSignTemplate(region string) *fasttemplate.Template {
	const strToSignTemplate = `AWS4-HMAC-SHA256
[time-long]
[time-short]/[region]/dynamodb/aws4_request
[canonical-string-hash]`

	precompiled := fasttemplate.ExecuteStringStd(strToSignTemplate, startTag, endTag,
		map[string]interface{}{
			tagRegion: region,
		})

	return fasttemplate.New(precompiled, startTag, endTag)
}

func getCanonicalStringTemplate(host string, credentials aws.Credentials) *fasttemplate.Template {
	const canonicalStringTemplate = `POST
/

accept-encoding:identity
content-length:[length]
content-type:application/x-amz-json-1.0
host:[host]
x-amz-date:[time-long][security-token-canonical-header]
x-amz-target:DynamoDB_20120810.GetItem

accept-encoding;content-length;content-type;host;x-amz-date;[security-token-header]x-amz-target
[payload-hash]`

	m := map[string]interface{}{
		tagHost:                         host,
		tagSecurityTokenCanonicalHeader: "",
		tagSecurityTokenHeader:          "",
	}

	if credentials.SessionToken != "" {
		m[tagSecurityTokenCanonicalHeader] = "\nx-amz-security-token:" + credentials.SessionToken
		m[tagSecurityTokenHeader] = amzSecurityTokenKey + ";"
	}

	precompiled := fasttemplate.ExecuteStringStd(canonicalStringTemplate, startTag, endTag, m)

	return fasttemplate.New(precompiled, startTag, endTag)
}

//nolint:lll // Breaking the template definition would make it harder to manage.
func getAuthHeaderTemplate(region string, credentials aws.Credentials) *fasttemplate.Template {
	const authHeaderTemplate = `AWS4-HMAC-SHA256 Credential=[access-key-id]/[time-short]/[region]/dynamodb/aws4_request, SignedHeaders=accept-encoding;content-length;content-type;host;x-amz-date;[security-token-header]x-amz-target, Signature=[signature]`

	m := map[string]interface{}{
		tagAccessKeyID:         credentials.AccessKeyID,
		tagRegion:              region,
		tagSecurityTokenHeader: "",
	}

	if credentials.SessionToken != "" {
		m[tagSecurityTokenHeader] = amzSecurityTokenKey + ";"
	}

	precompiled := fasttemplate.ExecuteStringStd(authHeaderTemplate, startTag, endTag, m)

	return fasttemplate.New(precompiled, startTag, endTag)
}
