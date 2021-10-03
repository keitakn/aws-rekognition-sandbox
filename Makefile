.PHONY: build clean deploy test lint format ci generate-mock

build:
	GOOS=linux GOARCH=amd64 go build -o bin/imagerecognition ./cmd/lambda/imagerecognition/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/detectfaces ./cmd/lambda/detectfaces/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/isacceptablecatimage ./cmd/lambda/isacceptablecatimage/main.go

clean:
	rm -rf ./bin

deploy: clean build
	npm run deploy

remove:
	npm run remove

test:
	go clean -testcache
	go test -p 1 -v $$(go list ./... | grep -v /node_modules/)

lint:
	go vet ./...
	golangci-lint run ./...

format:
	gofmt -l -s -w .
	goimports -w -l ./

ci: clean build lint
	go mod tidy && git diff -s --exit-code go.sum
	go clean -testcache
	go test -p 1 -v -covermode atomic -coverprofile=covprofile.out $$(go list ./... | grep -v /node_modules/)

generate-mock:
	mockgen -source=infrastructure/rekognition_client.go -destination mock/rekognition_client.go -package mock
	mockgen -source=infrastructure/s3_uploader.go -destination mock/s3_uploader.go -package mock
	mockgen -source=infrastructure/unique_id_generator.go -destination mock/unique_id_generator.go -package mock
