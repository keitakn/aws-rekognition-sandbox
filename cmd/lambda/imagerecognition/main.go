package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/keitakn/aws-rekognition-sandbox/infrastructure"
	"github.com/keitakn/aws-rekognition-sandbox/usecase/imagerecognition"
)

var imageRecognitionUseCase *imagerecognition.UseCase

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
	uploader := manager.NewUploader(s3Client)

	rekognitionClient := rekognition.NewFromConfig(cfg)

	imageRecognitionUseCase = &imagerecognition.UseCase{
		RekognitionClient: rekognitionClient,
		S3Uploader:        uploader,
		UniqueIdGenerator: &infrastructure.UuidGenerator{},
	}
}

type RequestBody struct {
	Image          string `json:"image"`
	ImageExtension string `json:"imageExtension"`
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

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var reqBody RequestBody
	if err := json.Unmarshal([]byte(req.Body), &reqBody); err != nil {
		statusCode := 400

		res := createErrorResponse(statusCode, "Bad Request")

		return res, err
	}

	res := imageRecognitionUseCase.ImageRecognition(
		ctx,
		imagerecognition.RequestBody{
			Image:          reqBody.Image,
			ImageExtension: reqBody.ImageExtension,
		},
	)

	if res.IsError {
		statusCode := 500
		resp := createErrorResponse(statusCode, res.ErrorBody.Message)

		return resp, nil
	}

	resBodyJson, _ := json.Marshal(res.OkBody)

	statusCode := 200
	resp := createApiGatewayV2Response(statusCode, resBodyJson)

	return resp, nil
}

func main() {
	lambda.Start(Handler)
}
