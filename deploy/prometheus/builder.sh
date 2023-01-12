#!/bin/bash

docker build . -t icsforth/prometheus

docker push icsforth/prometheus:latest