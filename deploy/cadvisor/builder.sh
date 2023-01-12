#!/bin/bash

docker build . -t icsforth/cadvisor

docker push icsforth/cadvisor:latest