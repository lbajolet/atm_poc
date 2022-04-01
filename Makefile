.PHONY: bin/server

bin/server: bin
	go build -o bin/server cmd/main.go

bin:
	mkdir bin
