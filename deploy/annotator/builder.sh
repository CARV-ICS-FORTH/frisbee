#!/bin/bash

docker build . -t icsforth/annotator --network host

docker push icsforth/annotator:latest
