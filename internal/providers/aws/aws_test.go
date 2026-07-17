package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestApplyS3ClientOptionsUsesPathStyleWhenEnabled(t *testing.T) {
	os.Setenv("AWS_S3_FORCE_PATH_STYLE", "true")
	defer os.Unsetenv("AWS_S3_FORCE_PATH_STYLE")

	options := s3.Options{}
	applyS3ClientOptions(&options)

	assert.True(t, options.UsePathStyle)
}

func TestApplyS3ClientOptionsUsesVirtualHostStyleByDefault(t *testing.T) {
	os.Unsetenv("AWS_S3_FORCE_PATH_STYLE")

	options := s3.Options{}
	applyS3ClientOptions(&options)

	assert.False(t, options.UsePathStyle)
}
