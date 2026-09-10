#!/usr/bin/env bash
cd app/ui/app
npm install --silent
npm run build
cd ../../..
go build -ldflags="-s -w" -o "EOC" ./app/cmd/app/

