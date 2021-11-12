run:
	go run main.go

build:
	go build -o beacon

full-build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o beacon

docker-build:
	docker build --target compile-stage --cache-from=beacon:compile-stage --tag beacon:compile-stage .
	docker build --target runtime-stage --cache-from=beacon:compile-stage --cache-from=beacon:latest --tag beacon:latest .