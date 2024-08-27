#!/usr/bin/env bash

#go build -o openkvm .

nohup openkvm > openkvm.log 2>&1 &
