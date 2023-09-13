.PHONY: all build run clean update list

run: # Runs the application locally
	go run main.go

build: # Build the go application
	go build main.go

all: # Runs build and run
	go run main.go
	go build main.go

clean: # Add missing and remove unused go modules
	go mod tidy

list: # List modules that are being used
	go install github.com/icholy/gomajor@latest | gomajor list

update: # Install and update go modules
	go get -u ./...

proto: # Generate proto files
	protoc --go_out=plugins=grpc:. *.proto
	protoc --go-grpc_out=grpc lineblocs.proto

test: # Runs all the tests in the application and returns if they passed or failed, along with a coverage percentage
	go install github.com/mfridman/tparse@latest | go mod tidy
	go test -json -cover ./... | tparse -all -pass
