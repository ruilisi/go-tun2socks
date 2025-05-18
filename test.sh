#!/usr/bin/env bash

ip=`nslookup baidu.com |  awk '/Non-authoritative answer:/ {found=1; next} found && /Address:/ {print $2}' | head -n 1`
sudo route add $ip 10.255.0.2
sleep 1
echo "curl -H "Host: www.baidu.com" http://$ip/"
curl -H "Host: www.baidu.com" http://$ip/
