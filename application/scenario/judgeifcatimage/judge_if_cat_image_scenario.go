package judgeifcatimage

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

type JudgeIfCatImageScenario struct {
	S3Client          infrastructure.S3Client
	RekognitionClient infrastructure.RekognitionClient
}

type JudgeIfCatImageRequest struct {
	TargetS3BucketName      string
	TargetS3ObjectKey       string
	TargetS3ObjectVersionId string
}

type IsCatImageResponse struct {
	IsCatImage  bool     `json:"isCatImage"`
	TypesOfCats []string `json:"typesOfCats"`
}

func (
	s *JudgeIfCatImageScenario,
) JudgeIfCatImage(
	ctx context.Context,
	req *JudgeIfCatImageRequest,
) (*IsCatImageResponse, error) {
	s3Object := &types.S3Object{
		Bucket:  aws.String(req.TargetS3BucketName),
		Name:    aws.String(req.TargetS3ObjectKey),
		Version: aws.String(req.TargetS3ObjectVersionId),
	}

	ext := s.extractImageExtension(req.TargetS3ObjectKey)
	if ext == "" {
		// 拡張子が取れないという事はこれ以上処理は出来ないので関数を終了させる
		return nil, errors.New("Not Allowed ImageExtension")
	}

	detectLabelsOutput, err := s.detectLabels(ctx, s3Object)
	if err != nil {
		return nil, errors.New("failed detectLabels")
	}

	// ねこ画像かどうかを判定する
	isCatImageResponse := s.isCatImage(detectLabelsOutput.Labels)

	return isCatImageResponse, nil
}

type CopyCatImageToDestinationBucketRequest struct {
	TriggerBucketName     string
	DestinationBucketName string
	TargetS3ObjectKey     string
}

func (
	s *JudgeIfCatImageScenario,
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

	err := s.copyS3Object(ctx, copySource, req.DestinationBucketName, uploadKey)
	if err != nil {
		return err
	}

	return nil
}

func (s *JudgeIfCatImageScenario) detectLabels(
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

	output, err := s.RekognitionClient.DetectLabels(ctx, input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (s *JudgeIfCatImageScenario) copyS3Object(
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

	_, err := s.S3Client.CopyObject(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

func (s *JudgeIfCatImageScenario) isCatImage(labels []types.Label) *IsCatImageResponse {
	isCatImageResponse := &IsCatImageResponse{
		IsCatImage: false,
	}

	for _, label := range labels {
		// ラベルにCatが含まれていて、かつConfidenceが閾値より大きい場合はねこの画像と見なす
		const confidenceThreshold = 90
		if *label.Name == "Cat" && *label.Confidence > confidenceThreshold {
			isCatImageResponse.IsCatImage = true
		}

		// ねこの種類を判別する為の処理
		// label.Parents に "Cat" が含まれていれば、そのラベルはねこの種類という事にしている
		// .e.g. test/images/abyssinian-cat.jpg の場合は {"isCatImage": true, "typesOfCats": ["Abyssinian"]}
		// .e.g. test/images/manx-cat.jpg の場合は {"isCatImage": true, "typesOfCats": ["Manx"]}
		for _, parent := range label.Parents {
			if *parent.Name == "Cat" {
				isCatImageResponse.TypesOfCats = append(isCatImageResponse.TypesOfCats, *label.Name)
			}
		}
	}

	return isCatImageResponse
}

func (s *JudgeIfCatImageScenario) extractImageExtension(fileName string) string {
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
