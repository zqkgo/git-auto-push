#! /bin/bash
go build -o git-auto-push .
nohup ./git-auto-push > git-auto-push.log 2>&1 & 