FROM alpine:latest

RUN apk add --no-cache zmap git go wget && \
    wget https://github.com/aspnmy/ollama-scanner/releases/download/v2.2/ollama-scanner-linux-amd64 -O /usr/local/bin/ollama-scanner && \
    chmod +x /usr/local/bin/ollama-scanner

