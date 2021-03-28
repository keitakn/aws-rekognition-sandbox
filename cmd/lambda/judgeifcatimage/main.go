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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var downloader *s3manager.Downloader
var uploader *s3manager.Uploader
var rekognitionSdk *rekognition.Rekognition

//nolint:gochecknoinits
func init() {
	region := os.Getenv("REGION")

	sess, err := createSession(region)
	if err != nil {
		// TODO ここでエラーが発生した場合、致命的な問題が起きているのでちゃんとしたログを出すように改修する
		log.Fatalln(err)
	}

	downloader = s3manager.NewDownloader(sess)
	uploader = s3manager.NewUploader(sess)
	rekognitionSdk = rekognition.New(sess)
}

func createSession(region string) (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		S3ForcePathStyle: aws.Bool(true),
		Region:           aws.String(region),
	})

	if err != nil {
		return nil, err
	}

	return sess, nil
}

func downloadFromS3(downloader *s3manager.Downloader, bucket string, key string) (f *os.File, err error) {
	tmpFile, _ := os.CreateTemp("/tmp", "tmp_img_")

	defer func() {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			// TODO ここでエラーが発生した場合、致命的な問題が起きているのでちゃんとしたログを出すように改修する
			log.Fatalln(err)
		}
	}()

	_, err = downloader.Download(
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

func uploadToS3(uploader *s3manager.Uploader, imgBytesBuffer *bytes.Buffer, bucket string, key string) error {
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Body:        imgBytesBuffer,
		ContentType: aws.String("image/jpeg"),
		Key:         aws.String(key),
	})

	if err != nil {
		return err
	}

	return nil
}

func detectLabels(rekognitionSdk *rekognition.Rekognition, imgBuffer []byte) (*rekognition.DetectLabelsOutput, error) {
	// 画像解析
	rekognitionImage := &rekognition.Image{
		Bytes: imgBuffer,
	}

	// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
	const maxLabels = int64(10)
	// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
	const minConfidence = float64(85)

	input := &rekognition.DetectLabelsInput{}
	input.SetImage(rekognitionImage)
	input.SetMaxLabels(maxLabels)
	input.SetMinConfidence(minConfidence)

	output, err := rekognitionSdk.DetectLabels(input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

type IsCatImageResult struct {
	IsCatImage  bool     `json:"isCatImage"`
	TypesOfCats []string `json:"typesOfCats"`
}

func isCatImage(labels []*rekognition.Label) *IsCatImageResult {
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

		img, err := downloadFromS3(downloader, bucket, key)
		if err != nil {
			return err
		}

		// 画像解析
		imgBuffer, err := io.ReadAll(img)
		if err != nil {
			return err
		}

		detectLabelsOutput, err := detectLabels(rekognitionSdk, imgBuffer)
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

		err = uploadToS3(uploader, imgBytesBuffer, os.Getenv("TRIGGER_BUCKET_NAME"), uploadKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
