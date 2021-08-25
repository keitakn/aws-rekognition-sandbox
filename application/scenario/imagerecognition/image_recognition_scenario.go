package imagerecognition

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/keitakn/aws-rekognition-sandbox/infrastructure"
)

type ImageRecognitionRequestBody struct {
	Image          string `json:"image"`
	ImageExtension string `json:"imageExtension"`
}

type ImageRecognitionResponseOkBody struct {
	Labels []types.Label `json:"labels"`
}

type ImageRecognitionResponseErrorBody struct {
	Message string `json:"message"`
}

type ImageRecognitionResponse struct {
	OkBody    *ImageRecognitionResponseOkBody
	IsError   bool
	ErrorBody *ImageRecognitionResponseErrorBody
}

type ImageRecognitionScenario struct {
	RekognitionClient infrastructure.RekognitionClient
	S3Uploader        infrastructure.S3Uploader
	UniqueIdGenerator infrastructure.UniqueIdGenerator
}

func (
	s *ImageRecognitionScenario,
) ImageRecognition(
	ctx context.Context,
	req ImageRecognitionRequestBody,
) *ImageRecognitionResponse {
	decodedImg, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		return &ImageRecognitionResponse{
			IsError: true,
			ErrorBody: &ImageRecognitionResponseErrorBody{
				Message: "Failed Decode Base64 Image",
			},
		}
	}

	uuid, err := s.UniqueIdGenerator.Generate()
	if err != nil {
		return &ImageRecognitionResponse{
			IsError: true,
			ErrorBody: &ImageRecognitionResponseErrorBody{
				Message: "Failed Generate UniqueId",
			},
		}
	}

	buffer := new(bytes.Buffer)
	buffer.Write(decodedImg)

	uploadKey := "tmp/" + uuid + req.ImageExtension
	err = s.uploadToS3(
		ctx,
		os.Getenv("TRIGGER_BUCKET_NAME"),
		buffer,
		s.decideS3ContentType(req.ImageExtension),
		uploadKey,
	)

	if err != nil {
		return &ImageRecognitionResponse{
			IsError: true,
			ErrorBody: &ImageRecognitionResponseErrorBody{
				Message: "Failed Upload To S3",
			},
		}
	}

	detectLabelsOutput, err := s.detectLabels(ctx, decodedImg)
	if err != nil {
		return &ImageRecognitionResponse{
			IsError: true,
			ErrorBody: &ImageRecognitionResponseErrorBody{
				Message: "Failed recognition",
			},
		}
	}

	return &ImageRecognitionResponse{
		OkBody: &ImageRecognitionResponseOkBody{
			Labels: detectLabelsOutput.Labels,
		},
		IsError: false,
	}
}

func (s *ImageRecognitionScenario) uploadToS3(
	ctx context.Context,
	bucket string,
	body *bytes.Buffer,
	contentType string,
	key string,
) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Body:        body,
		ContentType: aws.String(contentType),
		Key:         aws.String(key),
	}

	_, err := s.S3Uploader.Upload(ctx, input)

	if err != nil {
		return err
	}

	return nil
}

func (
	s *ImageRecognitionScenario,
) detectLabels(
	ctx context.Context,
	decodedImg []byte,
) (*rekognition.DetectLabelsOutput, error) {
	// 画像解析
	rekognitionImage := &types.Image{
		Bytes: decodedImg,
	}

	// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
	const maxLabels = int32(10)
	// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
	const minConfidence = float32(80)

	input := &rekognition.DetectLabelsInput{
		Image:         rekognitionImage,
		MaxLabels:     aws.Int32(maxLabels),
		MinConfidence: aws.Float32(minConfidence),
	}

	output, err := s.RekognitionClient.DetectLabels(ctx, input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (s *ImageRecognitionScenario) decideS3ContentType(ext string) string {
	contentType := ""

	switch ext {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	default:
		contentType = "image/jpeg"
	}

	return contentType
}
