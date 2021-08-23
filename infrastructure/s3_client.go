package infrastructure

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client interface {
	CopyObject(
		ctx context.Context,
		params *s3.CopyObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.CopyObjectOutput, error)
}
