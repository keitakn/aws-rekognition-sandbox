package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var uploader *manager.Uploader
var downloader *manager.Downloader
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

	s3Client := s3.NewFromConfig(cfg)
	uploader = manager.NewUploader(s3Client)
	downloader = manager.NewDownloader(s3Client)

	rekognitionClient = rekognition.NewFromConfig(cfg)
}

func downloadFromS3(
	ctx context.Context,
	downloader *manager.Downloader,
	bucket string,
	key string,
) (f *os.File, err error) {
	tmpFile, _ := os.CreateTemp("/tmp", "tmp_img_")

	defer func() {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			// TODO ここでエラーが発生した場合、致命的な問題が起きているのでちゃんとしたログを出すように改修する
			log.Fatalln(err)
		}
	}()

	_, err = downloader.Download(
		ctx,
		tmpFile,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)

	if err != nil {
		return nil, err
	}

	return tmpFile, err
}

func uploadToS3(
	ctx context.Context,
	uploader *manager.Uploader,
	imgBytesBuffer *bytes.Buffer,
	bucket string,
	key string,
) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Body:        imgBytesBuffer,
		ContentType: aws.String("image/jpeg"),
		Key:         aws.String(key),
	}

	_, err := uploader.Upload(ctx, input)

	if err != nil {
		return err
	}

	return nil
}

func detectLabels(
	ctx context.Context,
	rekognitionClient *rekognition.Client,
	imgBuffer []byte,
) (*rekognition.DetectLabelsOutput, error) {
	// 画像解析
	rekognitionImage := &types.Image{
		Bytes: imgBuffer,
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
		const confidenceThreshold = 96
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

func Handler(ctx context.Context, event events.S3Event) error {
	for _, record := range event.Records {
		// recordの中にイベント発生させたS3のBucket名やKeyが入っている
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key

		img, err := downloadFromS3(ctx, downloader, bucket, key)
		if err != nil {
			return err
		}

		// 画像解析
		imgBuffer, err := io.ReadAll(img)
		if err != nil {
			return err
		}

		detectLabelsOutput, err := detectLabels(ctx, rekognitionClient, imgBuffer)
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

		imgBytesBuffer := new(bytes.Buffer)
		imgBytesBuffer.Write(imgBuffer)

		err = uploadToS3(ctx, uploader, imgBytesBuffer, os.Getenv("TRIGGER_BUCKET_NAME"), uploadKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
