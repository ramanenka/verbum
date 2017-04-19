#!/usr/bin/env sh

docker build -t verbumby/web . || (echo 'could not build the image'; exit 1)
