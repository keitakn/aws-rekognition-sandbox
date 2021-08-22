package imagerecognitiontest

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/keitakn/aws-rekognition-sandbox/application"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/mock/gomock"
	"github.com/keitakn/aws-rekognition-sandbox/mock"
	"github.com/keitakn/aws-rekognition-sandbox/test"
)

func TestMain(m *testing.M) {
	status := m.Run()

	os.Exit(status)
}

//nolint:funlen
func TestHandler(t *testing.T) {
	t.Run("Successful fetch the image label", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)

		base64Img, err := test.EncodeImageToBase64("../../images/moko-cat.jpg")
		if err != nil {
			t.Fatal("Error failed to encodeImageToBase64", err)
		}

		decodedImg, err := test.DecodeImageFromBase64(base64Img)
		if err != nil {
			t.Fatal("Error failed to decodeImageFromBase64", err)
		}

		rekognitionImage := &types.Image{
			Bytes: decodedImg,
		}

		// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
		const maxLabels = int32(10)
		// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
		const minConfidence = float32(80)

		detectLabelsInput := &rekognition.DetectLabelsInput{
			Image:         rekognitionImage,
			MaxLabels:     aws.Int32(maxLabels),
			MinConfidence: aws.Float32(minConfidence),
		}

		confidenceExpected := float32(99.9)
		expectedFirstLabelName := "Cat"
		expectedSecondLabelName := "Chinchilla Silver"
		expectedFirstParentsName := "Animal"

		parents := []types.Parent{
			{Name: aws.String(expectedFirstParentsName)},
		}

		labels := []types.Label{
			{Confidence: aws.Float32(confidenceExpected), Name: aws.String(expectedFirstLabelName)},
			{Confidence: aws.Float32(confidenceExpected), Name: aws.String(expectedSecondLabelName), Parents: parents},
		}

		detectLabelsOutput := &rekognition.DetectLabelsOutput{
			Labels: labels,
		}

		ctx := context.Background()

		mockRekognitionClient.EXPECT().DetectLabels(ctx, detectLabelsInput).Return(detectLabelsOutput, nil)

		mockS3Uploader := mock.NewMockS3Uploader(ctrl)

		buffer := new(bytes.Buffer)
		buffer.Write(decodedImg)

		mockUuid := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		key := "tmp/" + mockUuid + ".jpg"

		s3PutObjectInput := &s3.PutObjectInput{
			Bucket:      aws.String(os.Getenv("TRIGGER_BUCKET_NAME")),
			Body:        buffer,
			ContentType: aws.String("image/jpeg"),
			Key:         aws.String(key),
		}

		s3UploadOutput := &manager.UploadOutput{
			Location: "https://exmple.s3.ap-northeast-1.amazonaws.com/" + key,
		}

		mockS3Uploader.EXPECT().Upload(ctx, s3PutObjectInput).Return(s3UploadOutput, nil)

		mockUniqueIdGenerator := mock.NewMockUniqueIdGenerator(ctrl)
		mockUniqueIdGenerator.EXPECT().Generate().Return(mockUuid, nil)

		scenario := application.ImageRecognitionScenario{
			RekognitionClient: mockRekognitionClient,
			S3Uploader:        mockS3Uploader,
			UniqueIdGenerator: mockUniqueIdGenerator,
		}

		req := application.ImageRecognitionRequestBody{
			Image:          base64Img,
			ImageExtension: ".jpg",
		}

		res := scenario.ImageRecognition(ctx, req)

		resFirstLabelName := *res.OkBody.Labels[0].Name
		if resFirstLabelName != expectedFirstLabelName {
			t.Error("\nActually: ", resFirstLabelName, "\nExpected: ", expectedFirstLabelName)
		}

		resSecondLabelName := *res.OkBody.Labels[1].Name
		if resSecondLabelName != expectedSecondLabelName {
			t.Error("\nActually: ", resSecondLabelName, "\nExpected: ", expectedSecondLabelName)
		}

		resFirstParentsName := *res.OkBody.Labels[1].Parents[0].Name
		if resFirstParentsName != expectedFirstParentsName {
			t.Error("\nActually: ", resFirstParentsName, "\nExpected: ", expectedFirstParentsName)
		}

		if res.IsError {
			t.Error("\nActually: ", res.IsError, "\nExpected: ", false)
		}
	})

	t.Run("Failure Generate UniqueId", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		base64Img, err := test.EncodeImageToBase64("../../images/moko-cat.jpg")
		if err != nil {
			t.Fatal("Error failed to encodeImageToBase64", err)
		}

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)

		mockS3Uploader := mock.NewMockS3Uploader(ctrl)

		mockUniqueIdGenerator := mock.NewMockUniqueIdGenerator(ctrl)
		mockUniqueIdGenerator.EXPECT().Generate().Return("", errors.New("failed Generate UUID"))

		scenario := application.ImageRecognitionScenario{
			RekognitionClient: mockRekognitionClient,
			S3Uploader:        mockS3Uploader,
			UniqueIdGenerator: mockUniqueIdGenerator,
		}

		req := application.ImageRecognitionRequestBody{
			Image:          base64Img,
			ImageExtension: ".jpg",
		}

		ctx := context.Background()
		res := scenario.ImageRecognition(ctx, req)

		if !res.IsError {
			t.Error("\nActually: ", res.IsError, "\nExpected: ", true)
		}

		expectedErrorMessage := "Failed Generate UniqueId"
		if res.ErrorBody.Message != expectedErrorMessage {
			t.Error("\nActually: ", res.ErrorBody.Message, "\nExpected: ", expectedErrorMessage)
		}
	})

	t.Run("Failure uploadToS3", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)

		base64Img, err := test.EncodeImageToBase64("../../images/moko-cat.jpg")
		if err != nil {
			t.Fatal("Error failed to encodeImageToBase64", err)
		}

		decodedImg, err := test.DecodeImageFromBase64(base64Img)
		if err != nil {
			t.Fatal("Error failed to decodeImageFromBase64", err)
		}

		mockS3Uploader := mock.NewMockS3Uploader(ctrl)

		buffer := new(bytes.Buffer)
		buffer.Write(decodedImg)

		mockUuid := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		key := "tmp/" + mockUuid + ".jpg"

		s3PutObjectInput := &s3.PutObjectInput{
			Bucket:      aws.String(os.Getenv("TRIGGER_BUCKET_NAME")),
			Body:        buffer,
			ContentType: aws.String("image/jpeg"),
			Key:         aws.String(key),
		}

		ctx := context.Background()
		mockS3Uploader.EXPECT().Upload(ctx, s3PutObjectInput).Return(nil, errors.New("failed upload to S3"))

		mockUniqueIdGenerator := mock.NewMockUniqueIdGenerator(ctrl)
		mockUniqueIdGenerator.EXPECT().Generate().Return(mockUuid, nil)

		scenario := application.ImageRecognitionScenario{
			RekognitionClient: mockRekognitionClient,
			S3Uploader:        mockS3Uploader,
			UniqueIdGenerator: mockUniqueIdGenerator,
		}

		req := application.ImageRecognitionRequestBody{
			Image:          base64Img,
			ImageExtension: ".jpg",
		}

		res := scenario.ImageRecognition(ctx, req)

		if !res.IsError {
			t.Error("\nActually: ", res.IsError, "\nExpected: ", true)
		}

		expectedErrorMessage := "Failed Upload To S3"
		if res.ErrorBody.Message != expectedErrorMessage {
			t.Error("\nActually: ", res.ErrorBody.Message, "\nExpected: ", expectedErrorMessage)
		}
	})
}
