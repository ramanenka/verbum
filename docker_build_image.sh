#!/usr/bin/env sh

(cd statics/ && npm i)

docker build -t verbumby/web . || (echo 'could not build the image'; exit 1)
