PROGRAM=mirrors_status
DOCKER_TARGET=hub.deepin.io/deepin/mirrors_status
DOCKER_BUILD_TARGET=${DOCKER_TARGET}.builder

.PHONY: build run

build: 
	go build -o ${PROGRAM} mirrors_status/cmd

run:
	go run cmd/main.go

docker:
	docker build -f deployments/Dockerfile --target builder -t ${DOCKER_BUILD_TARGET} .
	docker build -f deployments/Dockerfile -t ${DOCKER_TARGET} .

docker-push:
	docker push ${DOCKER_BUILD_TARGET}
	docker push ${DOCKER_TARGET}

clean:
	rm -rf ${PROGRAM}

rebuild: clean build
