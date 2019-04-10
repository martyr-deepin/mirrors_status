BinDir=$(shell pwd)/bin

run:
	go run cmd/init.go

bin:
	mkdir -p bin
	cd cmd/cdn-check; go build -race -v -o $(BinDir)/cdn-check
	cd cmd/push_to_influxdb; go build -v -o $(BinDir)/push_to_influxdb
	cd cmd; go build init.go

.PHONY: bin run

jenkins_bin:
	if [ -z "$(WORKSPACE)" ]; then exit 1; fi
	docker run --rm -e GOPROXY=https://goproxy.deepin.io -v $(WORKSPACE):/workspace -w /workspace songwentai/golang-go:1.11 bash -c "go install -mod=readonly -v ./...; mkdir -p bin; cp -v \`go env GOPATH\`/bin/* bin/; chown -R --reference=/workspace /workspace/bin"

docker_image: bin docker_image0

docker_image0:
	docker build -t hub.deepin.io/deepin/mirrors_status:0.0.1 .

jenkins_docker_image: jenkins_bin
	if [ -z "$(DOCKER_IMAGE_NAME)" ]; then exit 1; fi
	if [ -z "$(DOCKER_IMAGE_TAG)" ]; then exit 1; fi
	docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .
