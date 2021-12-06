# An ARG declared before a FROM is outside of a build stage, so it can’t be used in any instruction after a FROM.
# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG Repo=github.com/tikv/tikv
ARG PreProcess=""
ARG TargetBin=./target/debug/tikv-server


FROM rust:1.57 as builder
ARG Repo
ARG PreProcess
ARG TargetBin


RUN apt-get update &&  apt-get install -y  libelf-dev libdw-dev cmake gcc


# Get code and move to the directory
RUN git clone https://${Repo}.git code
WORKDIR code


# Export the flags needed to instrument the program to collect code coverage, and the flags needed to work around
# some Rust features that are incompatible with gcov-based instrumentation.
ENV  CARGO_INCREMENTAL=0
ENV  RUSTDOCFLAGS="-Cpanic=abort"
ENV  RUSTFLAGS=" ${RUSTFLAGS} -Zprofile -Ccodegen-units=1 -Copt-level=0 -Clink-dead-code -Coverflow-checks=off -Zpanic_abort_tests -Cpanic=abort"


# Install the Rust toolchain for the code
RUN rustup self update \
  && rustup set profile minimal \
  && rustup default $(cat "rust-toolchain") \
  && rustup component add llvm-tools-preview


# Build the instrumented binary
RUN cargo build

# Use a minimal container
FROM kcov/kcov 
ARG Repo
ARG TargetBin

RUN apt-get update && apt-get install -y git libiberty-dev lighttpd procps

# Get code and move to the directory
RUN git clone https://${Repo}.git code
WORKDIR code

COPY --from=builder /code/${TargetBin} ${TargetBin}

RUN lighttpd -f /etc/lighttpd/lighttpd.conf

ENTRYPOINT ["/kcov", "--exclude-pattern=/usr", "/var/www/html/htdocs/", "${TargetBin}"]
