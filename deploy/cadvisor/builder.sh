#!/bin/bash

docker build . -t icsforth/cadvisor --network=host

docker push icsforth/cadvisor:latest
