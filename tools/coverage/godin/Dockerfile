# An ARG declared before a FROM is outside of a build stage, so it can’t be used in any instruction after a FROM.
# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG Repo=github.com/tikv/pd 
ARG PreProcess="./scripts/embed-dashboard-ui.sh"
ARG TargetBin=./cmd/pd-server

FROM golang:1.17-alpine as builder

ARG Repo
ARG PreProcess
ARG TargetBin


RUN apk add --no-cache \
    git \
    make \
    bash \
    curl \
    gcc \
    g++


# Install Goc for building the instrumented binary
RUN curl -s -L "https://github.com/qiniu/goc/releases/latest" | sed -nE 's!.*"([^"]*-linux-amd64.tar.gz)".*!https://github.com\1!p' | xargs -n 1 curl -L  | tar -zx && chmod +x goc && mv goc /goc


# Get code and move to the directory
RUN git clone https://${Repo}.git code
WORKDIR code

# Cache dependencies
RUN GO111MODULE=on go mod download

# Pre-game stuff
RUN ${PreProcess}

# Build the instrumented binary
WORKDIR ${TargetBin}
RUN /goc build --output /instr_bin


#
# Move the instrumented binary and tols to a dedicated container
#
FROM alpine
ARG Repo

# Add lcov tools and a lightweight webserver for publishing the results
RUN apk add --no-cache -X http://dl-cdn.alpinelinux.org/alpine/edge/testing \
    lighttpd lcov 


COPY --from=builder /go/code /${Repo} 

COPY --from=builder /goc /goc

COPY --from=builder /instr_bin /instr_bin

# tool to convert go test coverage to lcov format
COPY --from=icsforth/gcov2lcov /gcov2lcov /gcov2lcov

# add a script for continuous Report.
# The script should run from /
WORKDIR /
COPY ./update.sh /

ENTRYPOINT ["/update.sh"]
