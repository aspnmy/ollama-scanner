#!/bin/bash
# -gateway-mac 必须传
# 如何获取网关mac地址：ip addr show | grep link/ether
# 或者输入ip addr show，查看你的访问外网的那个网卡的mac地址
./ollama_scanner_zmap-linux-amd64 -gateway-mac 72:70:f1:10:cc:f9 -input ./ip.txt -output ./custom.csv -no-bench -T 1
