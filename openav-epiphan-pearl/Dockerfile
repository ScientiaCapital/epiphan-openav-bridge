FROM golang:latest

COPY source /go/src

ENV GOPATH=

WORKDIR /go/src/microservice-framework
RUN rm -f go.mod go.sum
RUN go mod init github.com/Dartmouth-OpenAV/microservice-framework
RUN go mod tidy

WORKDIR /go
# Change the module path for each microservice
RUN go mod init github.com/Dartmouth-OpenAV/microservice-epiphan-pearl
RUN go mod edit -replace github.com/Dartmouth-OpenAV/microservice-framework=./src/microservice-framework
RUN go mod tidy

WORKDIR /go/src
RUN go get -u
RUN go build -o /go/bin/microservice

# Use this entrypoint for a fully functional docker image
ENTRYPOINT /go/bin/microservice
