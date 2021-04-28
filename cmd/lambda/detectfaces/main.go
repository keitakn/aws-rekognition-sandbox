package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
)

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

	rekognitionClient = rekognition.NewFromConfig(cfg)
}

type RequestBody struct {
	Image string `json:"image"`
}

type ResponseOkBody struct {
	FaceDetails interface{} `json:"faceDetails"`
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

func detectFaces(ctx context.Context, decodedImg []byte) (*rekognition.DetectFacesOutput, error) {
	// 画像解析
	rekognitionImage := &types.Image{
		Bytes: decodedImg,
	}

	input := &rekognition.DetectFacesInput{
		Image: rekognitionImage,
	}

	output, err := rekognitionClient.DetectFaces(ctx, input)
	if err != nil {
		return nil, err
	}

	return output, nil
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

	detectFacesOutput, err := detectFaces(ctx, decodedImg)
	if err != nil {
		statusCode := 500

		res := createErrorResponse(statusCode, "Failed detectFaces")

		return res, err
	}

	resBody := &ResponseOkBody{FaceDetails: detectFacesOutput.FaceDetails}
	resBodyJson, _ := json.Marshal(resBody)

	statusCode := 200
	res := createApiGatewayV2Response(statusCode, resBodyJson)

	return res, nil
}

func main() {
	lambda.Start(Handler)
}
