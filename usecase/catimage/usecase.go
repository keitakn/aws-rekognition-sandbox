package catimage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/keitakn/aws-rekognition-sandbox/infrastructure"
	"github.com/pkg/errors"
)

type UseCase struct {
	S3Client          infrastructure.S3Client
	RekognitionClient infrastructure.RekognitionClient
}

type Request struct {
	TargetS3BucketName      string
	TargetS3ObjectKey       string
	TargetS3ObjectVersionId string
}

type IsAcceptableCatImageResponse struct {
	IsAcceptableCatImage bool     `json:"isAcceptableCatImage"`
	TypesOfCats          []string `json:"typesOfCats"`
}

var (
	ErrNotAllowedImageExtension = errors.New("not allowed image extension")
	ErrUnexpected               = errors.New("unexpected error")
)

func (
	u *UseCase,
) IsAcceptableCatImage(
	ctx context.Context,
	req *Request,
) (*IsAcceptableCatImageResponse, error) {
	s3Object := &types.S3Object{
		Bucket:  aws.String(req.TargetS3BucketName),
		Name:    aws.String(req.TargetS3ObjectKey),
		Version: aws.String(req.TargetS3ObjectVersionId),
	}

	ext := u.extractImageExtension(req.TargetS3ObjectKey)
	if ext == "" {
		// 拡張子が取れないという事はこれ以上処理は出来ないので関数を終了させる
		return nil, errors.Wrap(ErrNotAllowedImageExtension, "image extension is empty")
	}

	detectLabelsOutput, err := u.detectLabels(ctx, s3Object)
	if err != nil {
		return nil, errors.Wrap(ErrUnexpected, err.Error())
	}

	// 受け入れ可能なねこ画像かどうかを判定する
	response := u.isAcceptableCatImage(detectLabelsOutput.Labels)

	return response, nil
}

type CopyCatImageToDestinationBucketRequest struct {
	TriggerBucketName     string
	DestinationBucketName string
	TargetS3ObjectKey     string
}

func (
	u *UseCase,
) CopyCatImageToDestinationBucket(
	ctx context.Context,
	req *CopyCatImageToDestinationBucketRequest,
) error {
	copySource := fmt.Sprintf(
		"%s/%s",
		req.TriggerBucketName,
		req.TargetS3ObjectKey,
	)

	uploadKey := "cat-images/" + strings.ReplaceAll(req.TargetS3ObjectKey, "tmp/", "")

	err := u.copyS3Object(ctx, copySource, req.DestinationBucketName, uploadKey)
	if err != nil {
		return errors.Wrap(err, "failed to UseCase.copyS3Object")
	}

	return nil
}

func (u *UseCase) detectLabels(
	ctx context.Context,
	s3Object *types.S3Object,
) (*rekognition.DetectLabelsOutput, error) {
	// 画像解析
	rekognitionImage := &types.Image{
		S3Object: s3Object,
	}

	// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
	const maxLabels = int32(10)
	// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
	const minConfidence = float32(85)

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

func (u *UseCase) copyS3Object(
	ctx context.Context,
	copySource string,
	toBucket string,
	uploadKey string,
) error {
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(toBucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(uploadKey),
	}

	_, err := u.S3Client.CopyObject(ctx, input)
	if err != nil {
		return errors.Wrap(err, "failed to S3Client.CopyObject")
	}

	return nil
}

func (u *UseCase) isAcceptableCatImage(labels []types.Label) *IsAcceptableCatImageResponse {
	response := &IsAcceptableCatImageResponse{
		IsAcceptableCatImage: false,
	}

	for _, label := range labels {
		// ラベルにCatが含まれていて、かつConfidenceが閾値より大きい場合は受け入れ可能なねこの画像と見なす
		const confidenceThreshold = 90
		if *label.Name == "Cat" && *label.Confidence > confidenceThreshold {
			response.IsAcceptableCatImage = true
		}

		// ねこの種類を判別する為の処理
		// label.Parents に "Cat" が含まれていれば、そのラベルはねこの種類という事にしている
		// .e.g. test/images/abyssinian-cat.jpg の場合は {"isAcceptableCatImage": true, "typesOfCats": ["Abyssinian"]}
		// .e.g. test/images/manx-cat.jpg の場合は {"isAcceptableCatImage": true, "typesOfCats": ["Manx"]}
		for _, parent := range label.Parents {
			if *parent.Name == "Cat" {
				response.TypesOfCats = append(response.TypesOfCats, *label.Name)
			}
		}
	}

	return response
}

func (u *UseCase) extractImageExtension(fileName string) string {
	// 許可されている画像拡張子
	allowedImageExtList := [...]string{".jpg", ".jpeg", ".png", ".webp"}

	ext := filepath.Ext(fileName)

	for _, v := range allowedImageExtList {
		if ext == v {
			return v
		}
	}

	return ""
}
