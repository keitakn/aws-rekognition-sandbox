package infrastructure

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
)

type RekognitionClient interface {
	DetectFaces(
		ctx context.Context,
		params *rekognition.DetectFacesInput,
		optFns ...func(*rekognition.Options),
	) (*rekognition.DetectFacesOutput, error)
}
