BINARY   := bicep-deployer
CMD_PATH := ./cmd/server

.PHONY: build run tidy clean

build:
	go build -o $(BINARY) $(CMD_PATH)

run:
	go run $(CMD_PATH)/main.go

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
