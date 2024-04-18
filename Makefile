PROJECT_NAME := snow-service
PORT := 8000

docker-build: name ?= snow
docker-build: version ?= v1.0
docker-build:
	echo "docker build  start..."
	docker build -t $(name):$(version) .

docker-start: name ?= snow
docker-start: version ?= v1.0
docker-start:
	echo  "docker run ..."
	docker run --name $(PROJECT_NAME) -d -p $(PORT):$(PORT) $(name):$(version)

docker-stop:
	echo "docker stop"
	docker stop $(PROJECT_NAME) && docker rm $(PROJECT_NAME)

