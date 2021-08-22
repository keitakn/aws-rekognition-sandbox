package infrastructure

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
)

type RekognitionClient interface {
	DetectLabels(
		ctx context.Context,
		params *rekognition.DetectLabelsInput,
		optFns ...func(*rekognition.Options),
	) (*rekognition.DetectLabelsOutput, error)
	DetectFaces(
		ctx context.Context,
		params *rekognition.DetectFacesInput,
		optFns ...func(*rekognition.Options),
	) (*rekognition.DetectFacesOutput, error)
}
