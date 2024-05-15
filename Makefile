build:
	go build -o bin/gobank

run:	build
		./bin/gobank

reset:	build
		./bin/gobank --purge --seed --exit

test:
	go test -v ./...