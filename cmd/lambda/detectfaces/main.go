package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/keitakn/aws-rekognition-sandbox/usecase/detectfaces"
)

var detectFacesUseCase *detectfaces.UseCase

//nolint:gochecknoinits
func init() {
	region := os.Getenv("REGION")

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		// TODO ここでエラーが発生した場合、致命的な問題が起きているのでちゃんとしたログを出すように改修する
		log.Fatalln(err)
	}

	rekognitionClient := rekognition.NewFromConfig(cfg)

	detectFacesUseCase = &detectfaces.UseCase{RekognitionClient: rekognitionClient}
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
	resBody := &detectfaces.ResponseErrorBody{Message: message}
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
	var reqBody detectfaces.RequestBody
	if err := json.Unmarshal([]byte(req.Body), &reqBody); err != nil {
		statusCode := 400

		res := createErrorResponse(statusCode, "Bad Request")

		return res, err
	}

	scenarioRes := detectFacesUseCase.DetectFaces(ctx, reqBody)
	if scenarioRes.IsError {
		statusCode := 500

		switch scenarioRes.ErrorBody.Message {
		case "Failed Decode Base64 Image":
			res := createErrorResponse(statusCode, scenarioRes.ErrorBody.Message)
			return res, nil
		case "Failed detectFaces":
			res := createErrorResponse(statusCode, scenarioRes.ErrorBody.Message)
			return res, nil
		default:
			res := createErrorResponse(statusCode, "Internal Server Error")
			return res, nil
		}
	}
	statusCode := 200

	resBodyJson, _ := json.Marshal(scenarioRes.OkBody)
	res := createApiGatewayV2Response(statusCode, resBodyJson)

	return res, nil
}

func main() {
	lambda.Start(Handler)
}
