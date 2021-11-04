all:
	echo "TODO"

run:
	go run main.go

local-build:
	go build -o lighthouse-server

full-build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o lighthouse-server

docker-build:
	docker build --target compile-stage --cache-from=lighthouse-server:compile-stage --tag lighthouse-server:compile-stage .
	docker build --target runtime-stage --cache-from=lighthouse-server:compile-stage --cache-from=lighthouse-server:latest --tag lighthouse-server:latest .