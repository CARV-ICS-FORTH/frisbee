#!/bin/bash

docker build . -t icsforth/prometheus -f prometheus.Dockerfile

docker push icsforth/prometheus:latest