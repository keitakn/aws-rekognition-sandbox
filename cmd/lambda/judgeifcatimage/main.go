package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/keitakn/aws-rekognition-sandbox/application/scenario/judgeifcatimage"
)

var scenario *judgeifcatimage.JudgeIfCatImageScenario

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

	rekognitionClient := rekognition.NewFromConfig(cfg)

	scenario = &judgeifcatimage.JudgeIfCatImageScenario{
		S3Client:          s3Client,
		RekognitionClient: rekognitionClient,
	}
}

func Handler(ctx context.Context, event events.S3Event) error {
	for _, record := range event.Records {
		// recordの中にイベント発生させたS3のBucket名やKeyが入っている
		judgeIfCatImageRequest := &judgeifcatimage.JudgeIfCatImageRequest{
			TargetS3BucketName:      record.S3.Bucket.Name,
			TargetS3ObjectKey:       record.S3.Object.Key,
			TargetS3ObjectVersionId: record.S3.Object.VersionID,
		}

		// ねこ画像かどうかを判定する
		isCatImageResponse, err := scenario.JudgeIfCatImage(ctx, judgeIfCatImageRequest)
		if err != nil {
			return err
		}

		// ねこ画像ではない場合、ここで処理を中断する
		if !isCatImageResponse.IsCatImage {
			continue
		}

		copyCatImageRequest := &judgeifcatimage.CopyCatImageToDestinationBucketRequest{
			// TriggerBucketName, DestinationBucketNameに同じ値が設定されているが、同じバケットの異なるディレクトリを使っているから
			// 実運用の際は別のバケットを指定したほうが良い
			TriggerBucketName:     os.Getenv("TRIGGER_BUCKET_NAME"),
			DestinationBucketName: os.Getenv("TRIGGER_BUCKET_NAME"),
			TargetS3ObjectKey:     judgeIfCatImageRequest.TargetS3ObjectKey,
		}

		// ここまで来るという事はねこ画像なので指定された場所にアップロードする
		err = scenario.CopyCatImageToDestinationBucket(ctx, copyCatImageRequest)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
