package application

import (
	"context"
	"encoding/base64"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/keitakn/aws-rekognition-sandbox/infrastructure"
)

type DetectFacesRequestBody struct {
	Image string `json:"image"`
}

type DetectFacesResponseOkBody struct {
	FaceDetails interface{} `json:"faceDetails"`
}

type DetectFacesResponseErrorBody struct {
	Message string `json:"message"`
}

type DetectFacesResponse struct {
	OkBody    *DetectFacesResponseOkBody
	IsError   bool
	ErrorBody *DetectFacesResponseErrorBody
}

type DetectFacesScenario struct {
	RekognitionClient infrastructure.RekognitionClient
}

func (s *DetectFacesScenario) DetectFaces(ctx context.Context, req DetectFacesRequestBody) *DetectFacesResponse {
	decodedImg, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		return &DetectFacesResponse{
			IsError:   true,
			ErrorBody: &DetectFacesResponseErrorBody{Message: "Failed Decode Base64 Image"},
		}
	}

	detectFacesOutput, err := s.detectFaces(ctx, decodedImg)
	if err != nil {
		return &DetectFacesResponse{
			IsError:   true,
			ErrorBody: &DetectFacesResponseErrorBody{Message: "Failed detectFaces"},
		}
	}

	return &DetectFacesResponse{
		OkBody: &DetectFacesResponseOkBody{
			FaceDetails: detectFacesOutput,
		},
		IsError: false,
	}
}

func (
	s *DetectFacesScenario,
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

	output, err := s.RekognitionClient.DetectFaces(ctx, input)
	if err != nil {
		return nil, err
	}

	return output, nil
}
