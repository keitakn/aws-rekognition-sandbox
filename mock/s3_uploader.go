// Code generated by MockGen. DO NOT EDIT.
// Source: infrastructure/s3_uploader.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	gomock "github.com/golang/mock/gomock"
)

// MockS3Uploader is a mock of S3Uploader interface.
type MockS3Uploader struct {
	ctrl     *gomock.Controller
	recorder *MockS3UploaderMockRecorder
}

// MockS3UploaderMockRecorder is the mock recorder for MockS3Uploader.
type MockS3UploaderMockRecorder struct {
	mock *MockS3Uploader
}

// NewMockS3Uploader creates a new mock instance.
func NewMockS3Uploader(ctrl *gomock.Controller) *MockS3Uploader {
	mock := &MockS3Uploader{ctrl: ctrl}
	mock.recorder = &MockS3UploaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockS3Uploader) EXPECT() *MockS3UploaderMockRecorder {
	return m.recorder
}

// Upload mocks base method.
func (m *MockS3Uploader) Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, input}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Upload", varargs...)
	ret0, _ := ret[0].(*manager.UploadOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Upload indicates an expected call of Upload.
func (mr *MockS3UploaderMockRecorder) Upload(ctx, input interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, input}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upload", reflect.TypeOf((*MockS3Uploader)(nil).Upload), varargs...)
}
