#!/bin/bash

rm main function.zip
GOARCH=amd64 GOOS=linux go build main.go
zip function.zip main
