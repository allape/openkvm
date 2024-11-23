#!/usr/bin/env bash

chmod +x ./openkvm

nohup ./openkvm > openkvm.log 2>&1 &
