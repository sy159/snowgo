PROJECT_NAME := snowgo-service
PORT := 8000

docker-build: name ?= snowgo
docker-build: version ?= v1.0
docker-build:
	echo "docker build  start..."
	docker build -t $(name):$(version) .

docker-start: name ?= snowgo
docker-start: version ?= v1.0
docker-start:
	echo  "docker run ..."
	docker run --name $(PROJECT_NAME) -d -p $(PORT):$(PORT) $(name):$(version)

docker-stop:
	echo "docker stop"
	docker stop $(PROJECT_NAME) && docker rm $(PROJECT_NAME)


# 生成model
.PHONY: gen
gen: do ?= init
gen:
	go run ./internal/dal/cmd/gen.go $(do) && make gen-query
	# git add ./internal/dal/
gen-query:
	go run ./internal/dal/cmd/gen.go query
