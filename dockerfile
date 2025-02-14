FROM alpine:latest

ADD ./Releases/v2.2-zmap/ollama-scanner-linux-amd64 /usr/local/bin/ollama-scanner
RUN apk add --no-cache zmap go  && \
    chmod +x /usr/local/bin/ollama-scanner

