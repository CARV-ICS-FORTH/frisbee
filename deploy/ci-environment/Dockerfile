FROM jrei/systemd-ubuntu:22.04

SHELL ["/bin/bash", "-c"]

RUN apt-get update

# Install snap and make it support --classic deployments
RUN apt-get install -y snapd sudo curl

RUN ln -s /var/lib/snapd/snap /snap

ENV PATH="${PATH}:/snap/bin"

ENV LC_ALL=C.UTF-8
ENV LANG=C.UTF-8
