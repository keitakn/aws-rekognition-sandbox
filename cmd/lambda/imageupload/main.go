package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type ResponseOkBody struct {
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

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	resBody := &ResponseOkBody{Message: "Hello Amazon Rekognition🐱"}
	resBodyJson, _ := json.Marshal(resBody)

	statusCode := 200
	res := createApiGatewayV2Response(statusCode, resBodyJson)

	return res, nil
}

func main() {
	lambda.Start(Handler)
}
