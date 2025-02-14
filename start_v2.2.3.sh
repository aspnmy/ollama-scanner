#!/bin/sh

# 启动 MongoDB
mongod --dbpath /ollama_scanner/db --logpath /var/log/mongodb/mongod.log --fork

# 等待 MongoDB 启动
while ! mongo --eval "db.version()" > /dev/null 2>&1; do
    echo "等待 MongoDB 启动..."
    sleep 1
done

# 启动 ollama_scanner
/usr/local/bin/ollama_scanner
