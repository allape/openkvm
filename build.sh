#!/usr/bin/env bash

export GOARCH=arm64
export GOOS=linux

go build -trimpath -o openkvm .
