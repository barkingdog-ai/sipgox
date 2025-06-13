`# 使用 ngrep 監控 SIP 封包，並顯示回應
sudo ngrep -d any -W byline port 5060`


# 檢查網路延遲
ping 192.168.11.210

# 檢查 UDP 連線
nc -uv 192.168.11.210 5060

# 顯示更詳細的封包資訊
sudo tcpdump -i any port 5060 -vv -w sip_capture.pcap

# 檢查防火牆規則
sudo iptables -L