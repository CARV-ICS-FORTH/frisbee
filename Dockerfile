# Build the Frisbee operator binary
FROM golang:1.19 as builder


ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOPROXY=direct
ENV GOSUMDB=off

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN  go mod download

# go env -w GOPROXY=direct
  #go env -w GOSUMDB=off

# Copy the go source
COPY ./cmd/manager/main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN go build -a -o /manager main.go

# Use alpine as minimal base image to package the Frisbee operator binary
# We use a non-root user setup.
#
# usage:
# $ docker build --build-arg "USER=someuser" . -t test
# $ docker run --rm -v ${HOME}/.kube/:/home/default/.kube test
FROM alpine

ARG USER=default
ENV HOME /home/$USER

# install sudo as root
RUN apk add --update sudo

# add new user
RUN adduser -D $USER \
        && echo "$USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER \
        && chmod 0440 /etc/sudoers.d/$USER

USER $USER
WORKDIR $HOME

COPY --from=builder /manager ./

ENTRYPOINT ["./manager"]
