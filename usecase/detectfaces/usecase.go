package detectfaces

import (
	"context"
	"encoding/base64"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/keitakn/aws-rekognition-sandbox/infrastructure"
	"github.com/pkg/errors"
)

type Request struct {
	Image string `json:"image"`
}

type Response struct {
	DetectFacesOutput *rekognition.DetectFacesOutput `json:"detectFacesOutput"`
}

type ResponseErrorBody struct {
	Message string `json:"message"`
}

var (
	ErrBase64Decode = errors.New("failed to base64 decode")
	ErrUnexpected   = errors.New("unexpected error")
)

type UseCase struct {
	RekognitionClient infrastructure.RekognitionClient
}

func (u *UseCase) DetectFaces(ctx context.Context, req Request) (*Response, error) {
	decodedImg, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		return nil, errors.Wrap(ErrBase64Decode, err.Error())
	}

	detectFacesOutput, err := u.detectFaces(ctx, decodedImg)
	if err != nil {
		return nil, errors.Wrap(ErrUnexpected, err.Error())
	}

	return &Response{
		DetectFacesOutput: detectFacesOutput,
	}, nil
}

func (
	u *UseCase,
) detectFaces(
	ctx context.Context,
	decodedImg []byte,
) (*rekognition.DetectFacesOutput, error) {
	// 画像解析
	rekognitionImage := &types.Image{
		Bytes: decodedImg,
	}

	input := &rekognition.DetectFacesInput{
		Image: rekognitionImage,
	}

	output, err := u.RekognitionClient.DetectFaces(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to RekognitionClient.DetectFaces")
	}

	return output, nil
}
