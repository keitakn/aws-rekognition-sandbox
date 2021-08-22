package test

import (
	"encoding/base64"
	"os"
)

func EncodeImageToBase64(imgPath string) (string, error) {
	bytes, err := os.ReadFile(imgPath)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}

func DecodeImageFromBase64(base64Img string) ([]byte, error) {
	decodedImg, err := base64.StdEncoding.DecodeString(base64Img)
	if err != nil {
		return nil, err
	}

	return decodedImg, nil
}
