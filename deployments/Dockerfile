FROM golang:1.12.5-stretch AS builder

RUN apt-get update
RUN apt-get install make
RUN apt-get install git
RUN apt-get install gcc libc-dev
RUN apt-get clean

ENV GOPATH=/opt/gopath
ENV PROJECT_PATH=${GOPATH}/src/mirrors_status
ENV GO111MODULE=off 

RUN mkdir -p ${PROJECT_PATH}
ADD . ${PROJECT_PATH}
WORKDIR ${PROJECT_PATH}
RUN make

FROM golang:1.12.5-stretch

RUN apt-get update
RUN apt-get install ca-certificates
RUN apt-get clean

ENV GOPATH=/opt/gopath
ENV PROJECT_PATH=${GOPATH}/src/mirrors_status

RUN mkdir -p ${PROJECT_PATH}
RUN mkdir -p ${PROJECT_PATH}/configs
WORKDIR ${PROJECT_PATH}
COPY --from=builder ${PROJECT_PATH}/mirrors_status ${PROJECT_PATH}/mirrors_status
COPY --from=builder ${PROJECT_PATH}/configs/config.example.yml ${PROJECT_PATH}/configs/config.yml
COPY --from=builder ${PROJECT_PATH}/configs/jenkins.example.yml ${PROJECT_PATH}/configs/jenkins.yml

CMD ["/opt/gopath/src/mirrors_status/mirrors_status"]
