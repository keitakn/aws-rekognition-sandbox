package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

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

	uploader = s3manager.NewUploader(sess)
	rekognitionSdk = rekognition.New(sess)
}

type RequestBody struct {
	Image string `json:"image"`
}

type ResponseOkBody struct {
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

type ResponseErrorBody struct {
	Message string `json:"message"`
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
	var reqBody RequestBody
	if err := json.Unmarshal([]byte(req.Body), &reqBody); err != nil {
		resBody := &ResponseErrorBody{Message: "Bad Request"}
		resBodyJson, _ := json.Marshal(resBody)

		statusCode := 400

		res := events.APIGatewayV2HTTPResponse{
			StatusCode: statusCode,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:            string(resBodyJson),
			IsBase64Encoded: false,
		}

		return res, err
	}

	decodeImg, err := base64.StdEncoding.DecodeString(reqBody.Image)

	uid, err := uuid.NewRandom()
	if err != nil {
		resBody := &ResponseErrorBody{Message: "Failed Generate UUID"}
		resBodyJson, _ := json.Marshal(resBody)

		statusCode := 500

		res := events.APIGatewayV2HTTPResponse{
			StatusCode: statusCode,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:            string(resBodyJson),
			IsBase64Encoded: false,
		}

		return res, err
	}

	buffer := new(bytes.Buffer)
	buffer.Write(decodeImg)

	uploadKey := "tmp/" + uid.String() + ".jpg"
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(os.Getenv("TRIGGER_BUCKET_NAME")),
		Body:        buffer,
		ContentType: aws.String("image/jpeg"),
		Key:         aws.String(uploadKey),
	})

	if err != nil {
		resBody := &ResponseErrorBody{Message: "Failed Upload To S3"}
		resBodyJson, _ := json.Marshal(resBody)

		statusCode := 500

		res := events.APIGatewayV2HTTPResponse{
			StatusCode: statusCode,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:            string(resBodyJson),
			IsBase64Encoded: false,
		}

		return res, err
	}

	// 画像解析
	image := &rekognition.Image{
		Bytes: decodeImg,
	}

	input := &rekognition.DetectLabelsInput{}
	input.SetImage(image)
	input.SetMaxLabels(10)
	input.SetMinConfidence(50)
	output, err := rekognitionSdk.DetectLabels(input)
	if err != nil {
		resBody := &ResponseErrorBody{Message: "Failed rekognition"}
		resBodyJson, _ := json.Marshal(resBody)

		statusCode := 500

		res := events.APIGatewayV2HTTPResponse{
			StatusCode: statusCode,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:            string(resBodyJson),
			IsBase64Encoded: false,
		}

		return res, err
	}

	resBody := &ResponseOkBody{Message: "Hello Amazon Rekognition🐱", Result: output.Labels}
	resBodyJson, _ := json.Marshal(resBody)

	statusCode := 200
	res := createApiGatewayV2Response(statusCode, resBodyJson)

	return res, nil
}

func main() {
	lambda.Start(Handler)
}
