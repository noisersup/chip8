BIN_NAME=chip8-go

build:
	go build -o ${BIN_NAME} main.go

run:
	go run main.go

clean:
	go clean
