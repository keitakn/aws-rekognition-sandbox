package judgeifcatimagetest

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/golang/mock/gomock"
	"github.com/keitakn/aws-rekognition-sandbox/application"
	"github.com/keitakn/aws-rekognition-sandbox/mock"
	"github.com/pkg/errors"
)

func TestMain(m *testing.M) {
	status := m.Run()

	os.Exit(status)
}

//nolint:funlen
func TestHandler(t *testing.T) {
	const expectedTriggerBucketName = "trigger-bucket"
	const expectedTargetS3ObjectVersionId = "AAAAA.1234567890123456789abcdefg"
	const catLabelName = "Cat"

	t.Run("judged to be a cat image", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)
		expectedTargetS3ObjectKey := "tmp/sample-cat-image.jpg"

		s3Object := &types.S3Object{
			Bucket:  aws.String(expectedTriggerBucketName),
			Name:    aws.String(expectedTargetS3ObjectKey),
			Version: aws.String(expectedTargetS3ObjectVersionId),
		}

		rekognitionImage := &types.Image{
			S3Object: s3Object,
		}

		// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
		const maxLabels = int32(10)
		// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
		const minConfidence = float32(85)

		detectLabelsInput := &rekognition.DetectLabelsInput{
			Image:         rekognitionImage,
			MaxLabels:     aws.Int32(maxLabels),
			MinConfidence: aws.Float32(minConfidence),
		}

		confidenceExpected := float32(90.1)
		expectedFirstLabelName := catLabelName
		expectedSecondLabelName := "ChinchillaSilver"
		expectedFirstParentsName := catLabelName

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

		scenario := application.JudgeIfCatImageScenario{
			RekognitionClient: mockRekognitionClient,
		}

		req := &application.JudgeIfCatImageRequest{
			TargetS3BucketName:      expectedTriggerBucketName,
			TargetS3ObjectKey:       expectedTargetS3ObjectKey,
			TargetS3ObjectVersionId: expectedTargetS3ObjectVersionId,
		}

		res, err := scenario.JudgeIfCatImage(ctx, req)
		if err != nil {
			t.Fatal("Failed JudgeIfCatImage", err)
		}

		expected := &application.IsCatImageResponse{
			IsCatImage:  true,
			TypesOfCats: []string{expectedSecondLabelName},
		}

		if reflect.DeepEqual(res, expected) == false {
			t.Error("\nActually: ", res, "\nExpected: ", expected)
		}
	})

	t.Run("judged that it is not a cat image, because the confidence value is low", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)
		expectedTargetS3ObjectKey := "tmp/sample-cat-image.jpg"

		s3Object := &types.S3Object{
			Bucket:  aws.String(expectedTriggerBucketName),
			Name:    aws.String(expectedTargetS3ObjectKey),
			Version: aws.String(expectedTargetS3ObjectVersionId),
		}

		rekognitionImage := &types.Image{
			S3Object: s3Object,
		}

		// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
		const maxLabels = int32(10)
		// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
		const minConfidence = float32(85)

		detectLabelsInput := &rekognition.DetectLabelsInput{
			Image:         rekognitionImage,
			MaxLabels:     aws.Int32(maxLabels),
			MinConfidence: aws.Float32(minConfidence),
		}

		confidenceExpected := float32(90.0)
		expectedFirstLabelName := catLabelName

		labels := []types.Label{
			{Confidence: aws.Float32(confidenceExpected), Name: aws.String(expectedFirstLabelName)},
		}

		detectLabelsOutput := &rekognition.DetectLabelsOutput{
			Labels: labels,
		}

		ctx := context.Background()

		mockRekognitionClient.EXPECT().DetectLabels(ctx, detectLabelsInput).Return(detectLabelsOutput, nil)

		scenario := application.JudgeIfCatImageScenario{
			RekognitionClient: mockRekognitionClient,
		}

		req := &application.JudgeIfCatImageRequest{
			TargetS3BucketName:      expectedTriggerBucketName,
			TargetS3ObjectKey:       expectedTargetS3ObjectKey,
			TargetS3ObjectVersionId: expectedTargetS3ObjectVersionId,
		}

		res, err := scenario.JudgeIfCatImage(ctx, req)
		if err != nil {
			t.Fatal("Failed JudgeIfCatImage", err)
		}

		expected := &application.IsCatImageResponse{
			IsCatImage: false,
		}

		if reflect.DeepEqual(res, expected) == false {
			t.Error("\nActually: ", res, "\nExpected: ", expected)
		}
	})

	t.Run("judged that it is not a cat image, because there is no cat in the image", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)
		expectedTargetS3ObjectKey := "tmp/sample-dog-image.jpg"

		s3Object := &types.S3Object{
			Bucket:  aws.String(expectedTriggerBucketName),
			Name:    aws.String(expectedTargetS3ObjectKey),
			Version: aws.String(expectedTargetS3ObjectVersionId),
		}

		rekognitionImage := &types.Image{
			S3Object: s3Object,
		}

		// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
		const maxLabels = int32(10)
		// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
		const minConfidence = float32(85)

		detectLabelsInput := &rekognition.DetectLabelsInput{
			Image:         rekognitionImage,
			MaxLabels:     aws.Int32(maxLabels),
			MinConfidence: aws.Float32(minConfidence),
		}

		confidenceExpected := float32(99.9)
		expectedFirstLabelName := "Dog"

		labels := []types.Label{
			{Confidence: aws.Float32(confidenceExpected), Name: aws.String(expectedFirstLabelName)},
		}

		detectLabelsOutput := &rekognition.DetectLabelsOutput{
			Labels: labels,
		}

		ctx := context.Background()

		mockRekognitionClient.EXPECT().DetectLabels(ctx, detectLabelsInput).Return(detectLabelsOutput, nil)

		scenario := application.JudgeIfCatImageScenario{
			RekognitionClient: mockRekognitionClient,
		}

		req := &application.JudgeIfCatImageRequest{
			TargetS3BucketName:      expectedTriggerBucketName,
			TargetS3ObjectKey:       expectedTargetS3ObjectKey,
			TargetS3ObjectVersionId: expectedTargetS3ObjectVersionId,
		}

		res, err := scenario.JudgeIfCatImage(ctx, req)
		if err != nil {
			t.Fatal("Failed JudgeIfCatImage", err)
		}

		expected := &application.IsCatImageResponse{
			IsCatImage: false,
		}

		if reflect.DeepEqual(res, expected) == false {
			t.Error("\nActually: ", res, "\nExpected: ", expected)
		}
	})

	t.Run("failure it is not an allowed image extension", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)
		expectedTargetS3ObjectKey := "tmp/sample-cat-image.gif"

		scenario := application.JudgeIfCatImageScenario{
			RekognitionClient: mockRekognitionClient,
		}

		req := &application.JudgeIfCatImageRequest{
			TargetS3BucketName:      expectedTriggerBucketName,
			TargetS3ObjectKey:       expectedTargetS3ObjectKey,
			TargetS3ObjectVersionId: expectedTargetS3ObjectVersionId,
		}

		ctx := context.Background()

		expected := "Not Allowed ImageExtension"
		_, err := scenario.JudgeIfCatImage(ctx, req)
		if err.Error() != expected {
			t.Error("\nActually: ", err.Error(), "\nExpected: ", expected)
		}
	})

	t.Run("failure because an error occurred in rekognitionClient", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRekognitionClient := mock.NewMockRekognitionClient(ctrl)
		expectedTargetS3ObjectKey := "tmp/sample-error-image.jpg"

		s3Object := &types.S3Object{
			Bucket:  aws.String(expectedTriggerBucketName),
			Name:    aws.String(expectedTargetS3ObjectKey),
			Version: aws.String(expectedTargetS3ObjectVersionId),
		}

		rekognitionImage := &types.Image{
			S3Object: s3Object,
		}

		// 何個までラベルを取得するかの設定、ラベルは信頼度が高い順に並んでいる
		const maxLabels = int32(10)
		// 信頼度の閾値、Confidenceがここで設定した値未満の場合、そのラベルはレスポンスに含まれない
		const minConfidence = float32(85)

		detectLabelsInput := &rekognition.DetectLabelsInput{
			Image:         rekognitionImage,
			MaxLabels:     aws.Int32(maxLabels),
			MinConfidence: aws.Float32(minConfidence),
		}

		ctx := context.Background()

		mockRekognitionClient.EXPECT().DetectLabels(
			ctx,
			detectLabelsInput,
		).Return(
			nil,
			errors.New("failed rekognitionClient detectLabels"),
		)

		scenario := application.JudgeIfCatImageScenario{
			RekognitionClient: mockRekognitionClient,
		}

		req := &application.JudgeIfCatImageRequest{
			TargetS3BucketName:      expectedTriggerBucketName,
			TargetS3ObjectKey:       expectedTargetS3ObjectKey,
			TargetS3ObjectVersionId: expectedTargetS3ObjectVersionId,
		}

		expected := "failed detectLabels"
		_, err := scenario.JudgeIfCatImage(ctx, req)
		if err.Error() != expected {
			t.Error("\nActually: ", err.Error(), "\nExpected: ", expected)
		}
	})
}
