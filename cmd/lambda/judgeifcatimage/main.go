package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client
var rekognitionClient *rekognition.Client

//nolint:gochecknoinits
func init() {
	region := os.Getenv("REGION")

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		// TODO ここでエラーが発生した場合、致命的な問題が起きているのでちゃんとしたログを出すように改修する
		log.Fatalln(err)
	}

	s3Client = s3.NewFromConfig(cfg)

	rekognitionClient = rekognition.NewFromConfig(cfg)
}

func detectLabels(
	ctx context.Context,
	rekognitionClient *rekognition.Client,
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

	output, err := rekognitionClient.DetectLabels(ctx, input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func copyS3Object(
	ctx context.Context,
	s3Client *s3.Client,
	copySource string,
	toBucket string,
	uploadKey string,
) error {
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(toBucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(uploadKey),
	}

	_, err := s3Client.CopyObject(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

type IsCatImageResult struct {
	IsCatImage  bool     `json:"isCatImage"`
	TypesOfCats []string `json:"typesOfCats"`
}

func isCatImage(labels []types.Label) *IsCatImageResult {
	isCatImageResult := &IsCatImageResult{
		IsCatImage: false,
	}

	for _, label := range labels {
		// ラベルにCatが含まれていて、かつConfidenceが閾値より大きい場合はねこの画像と見なす
		const confidenceThreshold = 90
		if *label.Name == "Cat" && *label.Confidence > confidenceThreshold {
			isCatImageResult.IsCatImage = true
		}

		// ねこの種類を判別する為の処理
		// label.Parents に "Cat" が含まれていれば、そのラベルはねこの種類という事にしている
		// .e.g. test/images/abyssinian-cat.jpg の場合は {"isCatImage": true, "typesOfCats": ["Abyssinian"]}
		// .e.g. test/images/manx-cat.jpg の場合は {"isCatImage": true, "typesOfCats": ["Manx"]}
		for _, parent := range label.Parents {
			if *parent.Name == "Cat" {
				isCatImageResult.TypesOfCats = append(isCatImageResult.TypesOfCats, *label.Name)
			}
		}
	}

	return isCatImageResult
}

func extractImageExtension(fileName string) string {
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

func Handler(ctx context.Context, event events.S3Event) error {
	for _, record := range event.Records {
		// recordの中にイベント発生させたS3のBucket名やKeyが入っている
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key

		s3Object := &types.S3Object{
			Bucket:  aws.String(bucket),
			Name:    aws.String(key),
			Version: aws.String(record.S3.Object.VersionID),
		}

		ext := extractImageExtension(key)
		if ext == "" {
			// 拡張子が取れないという事はこれ以上処理は出来ないので関数を終了させる
			return nil
		}

		detectLabelsOutput, err := detectLabels(ctx, rekognitionClient, s3Object)
		if err != nil {
			return err
		}

		// ねこ画像かどうかを判定する
		isCatImageResult := isCatImage(detectLabelsOutput.Labels)

		// ねこ画像ではない場合、ここで処理を中断する
		if !isCatImageResult.IsCatImage {
			continue
		}

		uploadKey := "cat-images/" + strings.ReplaceAll(key, "tmp/", "")

		copySource := fmt.Sprintf(
			"%s/%s",
			os.Getenv("TRIGGER_BUCKET_NAME"),
			key,
		)

		err = copyS3Object(
			ctx,
			s3Client,
			copySource,
			os.Getenv("TRIGGER_BUCKET_NAME"),
			uploadKey,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
