.PHONY: build test clean

build:
	go build -o bin/try main.go

test:
	go test -cover -count=1

clean:
	rm -rf bin/ dist/
