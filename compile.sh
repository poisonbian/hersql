#!/bin/bash
go clean
GOOS=windows GOARCH=amd64 go build -o hersql_win.exe
GOOS=darwin GOARCH=amd64 go build -o hersql_mac
GOOS=linux GOARCH=amd64 go build -o hersql_linux

