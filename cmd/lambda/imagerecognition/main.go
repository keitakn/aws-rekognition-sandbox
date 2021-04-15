package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var uploader *manager.Uploader
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

	rekognitionClient = rekognition.NewFromConfig(cfg)
}

type RequestBody struct {
	Image          string `json:"image"`
	ImageExtension string `json:"imageExtension"`
}

type ResponseOkBody struct {
	Labels interface{} `json:"labels"`
}

type ResponseErrorBody struct {
	Message string `json:"message"`
}

func createApiGatewayV2Response(statusCode int, resBodyJson []byte) events.APIGatewayV2HTTPResponse {
	res := events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            string(resBodyJson),
		IsBase64Encoded: false,
	}

	return res
}

func createErrorResponse(statusCode int, message string) events.APIGatewayV2HTTPResponse {
	resBody := &ResponseErrorBody{Message: message}
	resBodyJson, _ := json.Marshal(resBody)

	res := events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            string(resBodyJson),
		IsBase64Encoded: false,
	}

	return res
}

func uploadToS3(
	ctx context.Context,
	uploader *manager.Uploader,
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

	_, err := uploader.Upload(ctx, input)

	if err != nil {
		return err
	}

	return nil
}

func detectLabels(ctx context.Context, decodedImg []byte) (*rekognition.DetectLabelsOutput, error) {
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

	output, err := rekognitionClient.DetectLabels(ctx, input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func decideS3ContentType(ext string) string {
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

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var reqBody RequestBody
	if err := json.Unmarshal([]byte(req.Body), &reqBody); err != nil {
		statusCode := 400

		res := createErrorResponse(statusCode, "Bad Request")

		return res, err
	}

	decodedImg, err := base64.StdEncoding.DecodeString(reqBody.Image)
	if err != nil {
		statusCode := 500

		res := createErrorResponse(statusCode, "Failed Decode Base64 Image")

		return res, err
	}

	uid, err := uuid.NewRandom()
	if err != nil {
		statusCode := 500

		res := createErrorResponse(statusCode, "Failed Generate UUID")

		return res, err
	}

	buffer := new(bytes.Buffer)
	buffer.Write(decodedImg)

	uploadKey := "tmp/" + uid.String() + reqBody.ImageExtension
	err = uploadToS3(
		ctx,
		uploader,
		os.Getenv("TRIGGER_BUCKET_NAME"),
		buffer,
		decideS3ContentType(reqBody.ImageExtension),
		uploadKey,
	)

	if err != nil {
		statusCode := 500

		res := createErrorResponse(statusCode, "Failed Upload To S3")

		return res, err
	}

	detectLabelsOutput, err := detectLabels(ctx, decodedImg)
	if err != nil {
		statusCode := 500

		res := createErrorResponse(statusCode, "Failed recognition")

		return res, err
	}

	resBody := &ResponseOkBody{Labels: detectLabelsOutput.Labels}
	resBodyJson, _ := json.Marshal(resBody)

	statusCode := 200
	res := createApiGatewayV2Response(statusCode, resBodyJson)

	return res, nil
}

func main() {
	lambda.Start(Handler)
}
