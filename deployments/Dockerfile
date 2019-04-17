FROM golang:1.11.8-alpine3.8 AS builder

RUN apk add --update make
RUN apk add git
RUN apk add gcc libc-dev
RUN rm -rf /var/cache/apk/*

ENV GOPATH=/opt/gopath
ENV PROJECT_PATH=${GOPATH}/src/mirrors_status
ENV GO111MODULE=off 

RUN mkdir -p ${PROJECT_PATH}
ADD . ${PROJECT_PATH}
WORKDIR ${PROJECT_PATH}
RUN make

FROM golang:1.11.8-alpine3.8

RUN apk add --update ca-certificates
RUN rm -rf /var/cache/apk/*

ENV GOPATH=/opt/gopath
ENV PROJECT_PATH=${GOPATH}/src/mirrors_status

RUN mkdir -p ${PROJECT_PATH}
RUN mkdir -p ${PROJECT_PATH}/configs
WORKDIR ${PROJECT_PATH}
COPY --from=builder ${PROJECT_PATH}/mirrors_status ${PROJECT_PATH}/mirrors_status
COPY --from=builder ${PROJECT_PATH}/configs/config.example.yml ${PROJECT_PATH}/configs/config.yml

CMD ["/opt/gopath/src/mirrors_status/mirrors_status"]
