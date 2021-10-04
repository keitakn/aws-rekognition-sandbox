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
	"github.com/pkg/errors"
)

type RequestBody struct {
	Image          string `json:"image"`
	ImageExtension string `json:"imageExtension"`
}

type Response struct {
	Labels []types.Label `json:"labels"`
}

type UseCase struct {
	RekognitionClient infrastructure.RekognitionClient
	S3Uploader        infrastructure.S3Uploader
	UniqueIdGenerator infrastructure.UniqueIdGenerator
}

var (
	ErrBase64Decode     = errors.New("failed to base64 decode")
	ErrGenerateUniqueId = errors.New("failed to generate uniqueId")
	ErrUploadToS3       = errors.New("failed to upload to s3")
	ErrRekognition      = errors.New("failed to rekognition detectLabels")
)

func (
	u *UseCase,
) ImageRecognition(
	ctx context.Context,
	req RequestBody,
) (*Response, error) {
	decodedImg, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		return nil, errors.Wrap(ErrBase64Decode, err.Error())
	}

	uuid, err := u.UniqueIdGenerator.Generate()
	if err != nil {
		return nil, errors.Wrap(ErrGenerateUniqueId, err.Error())
	}

	buffer := new(bytes.Buffer)
	buffer.Write(decodedImg)

	uploadKey := "tmp/" + uuid + req.ImageExtension
	err = u.uploadToS3(
		ctx,
		os.Getenv("TRIGGER_BUCKET_NAME"),
		buffer,
		u.decideS3ContentType(req.ImageExtension),
		uploadKey,
	)

	if err != nil {
		return nil, errors.Wrap(ErrUploadToS3, err.Error())
	}

	detectLabelsOutput, err := u.detectLabels(ctx, decodedImg)
	if err != nil {
		return nil, errors.Wrap(ErrRekognition, err.Error())
	}

	return &Response{
		Labels: detectLabelsOutput.Labels,
	}, nil
}

func (u *UseCase) uploadToS3(
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

	_, err := u.S3Uploader.Upload(ctx, input)

	if err != nil {
		return errors.Wrap(err, "failed to S3Uploader.Upload")
	}

	return nil
}

func (
	u *UseCase,
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

	output, err := u.RekognitionClient.DetectLabels(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to RekognitionClient.DetectLabels")
	}

	return output, nil
}

func (u *UseCase) decideS3ContentType(ext string) string {
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
