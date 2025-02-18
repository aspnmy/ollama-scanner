#!/bin/bash 
# 清理缓存
go clean -modcache
# 重新同步
go mod download
go mod verify
go mod vendor