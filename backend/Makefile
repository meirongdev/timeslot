BINARY  := timeslot
CONFIG  ?= config.json

.PHONY: build run tidy clean

build:
	go build -o $(BINARY) .

run:
	go run . -config=$(CONFIG)

tidy:
	go mod tidy

clean:
	rm -f $(BINARY) timeslot.db
