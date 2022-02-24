#!/bin/sh -x
exec go build -ldflags="-s -w"
