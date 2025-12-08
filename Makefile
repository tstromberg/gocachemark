.PHONY: all build run hitrate latency throughput html clean lint deps

all: build

build:
	go build -o gocachemark .

run: build
	./gocachemark -all

hitrate: build
	./gocachemark -hitrate

latency: build
	./gocachemark -latency

throughput: build
	./gocachemark -throughput

html: build
	./gocachemark -all -html results.html
	@echo "Open results.html in a browser to view charts"

clean:
	rm -f gocachemark results.html

lint:
	golangci-lint run ./...

deps:
	go mod tidy
