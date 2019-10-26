#!/bin/sh

go build -o webdav-downloader main.go
GOOS=windows GOARCH=amd64 go build -o webdav-downloader.exe main.go
