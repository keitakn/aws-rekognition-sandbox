package detectfaces

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
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
	t.Run("Successful Face labels are detected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock.NewMockRekognitionClient(ctrl)

		base64Img, err := test.EncodeImageToBase64("../../test/images/moko-cat.jpg")
		if err != nil {
			t.Fatal("Error failed to encodeImageToBase64", err)
		}

		decodedImg, err := test.DecodeImageFromBase64(base64Img)
		if err != nil {
			t.Fatal("Error failed to decodeImageFromBase64", err)
		}

		params := &rekognition.DetectFacesInput{
			Image: &types.Image{Bytes: decodedImg},
		}

		confidenceExpected := float32(12.7)
		faceDetails := []types.FaceDetail{{Confidence: &confidenceExpected}}

		expectedDetectFacesOutput := &rekognition.DetectFacesOutput{
			FaceDetails: faceDetails,
		}

		ctx := context.Background()

		mockClient.EXPECT().DetectFaces(ctx, params).Return(expectedDetectFacesOutput, nil)

		req := &Request{
			Image: base64Img,
		}

		u := &UseCase{
			RekognitionClient: mockClient,
		}

		res, err := u.DetectFaces(ctx, *req)
		if err != nil {
			t.Fatal("Error failed to DetectFaces", err)
		}

		expected := &Response{
			DetectFacesOutput: expectedDetectFacesOutput,
		}

		resConfidence := *res.DetectFacesOutput.FaceDetails[0].Confidence
		if resConfidence != confidenceExpected {
			t.Error("\nActually: ", resConfidence, "\nExpected: ", confidenceExpected)
		}

		if reflect.DeepEqual(res, expected) == false {
			t.Error("\nActually: ", res, "\nExpected: ", expected)
		}
	})

	t.Run("Successful Face labels are not detected because it is not a human face", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock.NewMockRekognitionClient(ctrl)

		base64Img, err := test.EncodeImageToBase64("../../test/images/munchkin-cat.png")
		if err != nil {
			t.Fatal("Error failed to encodeImageToBase64", err)
		}

		decodedImg, err := test.DecodeImageFromBase64(base64Img)
		if err != nil {
			t.Fatal("Error failed to decodeImageFromBase64", err)
		}

		params := &rekognition.DetectFacesInput{
			Image: &types.Image{Bytes: decodedImg},
		}

		faceDetails := []types.FaceDetail{}

		expectedDetectFacesOutput := &rekognition.DetectFacesOutput{
			FaceDetails: faceDetails,
		}

		ctx := context.Background()

		mockClient.EXPECT().DetectFaces(ctx, params).Return(expectedDetectFacesOutput, nil)

		req := &Request{
			Image: base64Img,
		}

		u := &UseCase{
			RekognitionClient: mockClient,
		}

		res, err := u.DetectFaces(ctx, *req)
		if err != nil {
			t.Fatal("Error failed to DetectFaces", err)
		}

		expected := &Response{
			DetectFacesOutput: expectedDetectFacesOutput,
		}

		resFaceDetails := res.DetectFacesOutput.FaceDetails
		if len(resFaceDetails) != 0 {
			t.Error("\nActually: ", resFaceDetails)
		}

		if reflect.DeepEqual(res, expected) == false {
			t.Error("\nActually: ", res, "\nExpected: ", expected)
		}
	})

	t.Run("Failure DetectFaces returned an error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock.NewMockRekognitionClient(ctrl)

		base64Img, err := test.EncodeImageToBase64("../../test/images/munchkin-cat.png")
		if err != nil {
			t.Fatal("Error failed to encodeImageToBase64", err)
		}

		decodedImg, err := test.DecodeImageFromBase64(base64Img)
		if err != nil {
			t.Fatal("Error failed to decodeImageFromBase64", err)
		}

		params := &rekognition.DetectFacesInput{
			Image: &types.Image{Bytes: decodedImg},
		}

		ctx := context.Background()
		expectedDetectError := errors.New("DetectFaces Error")

		mockClient.EXPECT().DetectFaces(ctx, params).Return(nil, expectedDetectError)

		req := &Request{
			Image: base64Img,
		}

		u := &UseCase{
			RekognitionClient: mockClient,
		}

		_, err = u.DetectFaces(ctx, *req)
		expected := ErrUnexpected
		if !errors.Is(err, expected) {
			t.Error("\nActually: ", err, "\nExpected: ", expected)
		}
	})
}
