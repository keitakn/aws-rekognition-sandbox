package detectfaces

import (
	"context"
	"encoding/base64"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/keitakn/aws-rekognition-sandbox/infrastructure"
)

type RequestBody struct {
	Image string `json:"image"`
}

type ResponseOkBody struct {
	DetectFacesOutput *rekognition.DetectFacesOutput `json:"detectFacesOutput"`
}

type ResponseErrorBody struct {
	Message string `json:"message"`
}

type Response struct {
	OkBody    *ResponseOkBody
	IsError   bool
	ErrorBody *ResponseErrorBody
}

type UseCase struct {
	RekognitionClient infrastructure.RekognitionClient
}

func (u *UseCase) DetectFaces(ctx context.Context, req RequestBody) *Response {
	decodedImg, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		return &Response{
			IsError:   true,
			ErrorBody: &ResponseErrorBody{Message: "Failed Decode Base64 Image"},
		}
	}

	detectFacesOutput, err := u.detectFaces(ctx, decodedImg)
	if err != nil {
		return &Response{
			IsError:   true,
			ErrorBody: &ResponseErrorBody{Message: "Failed detectFaces"},
		}
	}

	return &Response{
		OkBody: &ResponseOkBody{
			DetectFacesOutput: detectFacesOutput,
		},
		IsError: false,
	}
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
		return nil, err
	}

	return output, nil
}
