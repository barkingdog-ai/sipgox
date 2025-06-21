#!/bin/bash

# SIP 連線測試腳本 - 診斷網路問題

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

SIP_SERVER="192.168.11.210"
SIP_PORT="5060"
CLIENT_IP="192.168.11.204"

echo -e "${PURPLE}SIP 連線診斷工具${NC}"
echo -e "${PURPLE}===============${NC}\n"

echo -e "${CYAN}1. 檢查網路連通性...${NC}"
if ping -c 3 "$SIP_SERVER" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 可以 ping 到 SIP 伺服器 ($SIP_SERVER)${NC}"
else
    echo -e "${RED}✗ 無法 ping 到 SIP 伺服器 ($SIP_SERVER)${NC}"
    echo -e "${YELLOW}  請檢查網路連線或 IP 設定${NC}"
fi

echo -e "\n${CYAN}2. 檢查端口連通性...${NC}"
if nc -zv "$SIP_SERVER" "$SIP_PORT" 2>&1 | grep -q "succeeded"; then
    echo -e "${GREEN}✓ SIP 端口 ($SIP_PORT) 可以連通${NC}"
else
    echo -e "${RED}✗ SIP 端口 ($SIP_PORT) 無法連通${NC}"
    echo -e "${YELLOW}  可能的原因:${NC}"
    echo -e "${YELLOW}  - 防火牆阻擋${NC}"
    echo -e "${YELLOW}  - SIP 伺服器未啟動${NC}"
    echo -e "${YELLOW}  - 端口設定錯誤${NC}"
fi

echo -e "\n${CYAN}3. 檢查本地端口占用...${NC}"
if netstat -ulpn | grep ":5060 " > /dev/null; then
    echo -e "${GREEN}✓ 本地端口 5060 有程式在監聽${NC}"
    netstat -ulpn | grep ":5060 " | while read line; do
        echo -e "${CYAN}  監聽詳情: $line${NC}"
    done
else
    echo -e "${YELLOW}⚠ 本地端口 5060 沒有程式在監聽${NC}"
fi

echo -e "\n${CYAN}4. 檢查防火牆設定...${NC}"
if command -v ufw > /dev/null; then
    if ufw status | grep -q "Status: active"; then
        echo -e "${YELLOW}⚠ UFW 防火牆已啟用${NC}"
        if ufw status | grep -q "5060"; then
            echo -e "${GREEN}✓ 防火牆已允許 SIP 端口${NC}"
        else
            echo -e "${RED}✗ 防火牆未允許 SIP 端口${NC}"
            echo -e "${YELLOW}  建議執行: sudo ufw allow 5060/udp${NC}"
        fi
    else
        echo -e "${CYAN}i UFW 防火牆未啟用${NC}"
    fi
else
    echo -e "${CYAN}i 未找到 UFW 防火牆${NC}"
fi

echo -e "\n${CYAN}5. 測試 UDP 封包傳送...${NC}"
echo -e "${YELLOW}發送測試 SIP OPTIONS 封包...${NC}"

# 建立測試 SIP OPTIONS 封包
SIP_OPTIONS="OPTIONS sip:test@$SIP_SERVER:$SIP_PORT SIP/2.0
Via: SIP/2.0/UDP $CLIENT_IP:5060;branch=z9hG4bK-test-123
From: <sip:test@$CLIENT_IP>;tag=test-from-tag
To: <sip:test@$SIP_SERVER>
Call-ID: test-call-id@$CLIENT_IP
CSeq: 1 OPTIONS
Max-Forwards: 70
User-Agent: SIPTestScript/1.0
Content-Length: 0

"

# 使用 nc 發送 SIP 封包並等待回應
echo "$SIP_OPTIONS" | timeout 5s nc -u "$SIP_SERVER" "$SIP_PORT"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 成功發送 SIP OPTIONS 封包${NC}"
else
    echo -e "${RED}✗ 發送 SIP OPTIONS 封包失敗或超時${NC}"
fi

echo -e "\n${CYAN}6. 網路介面資訊...${NC}"
echo -e "${CYAN}本地 IP 位址:${NC}"
ip addr show | grep "inet " | grep -v "127.0.0.1" | while read line; do
    echo -e "${CYAN}  $line${NC}"
done

echo -e "\n${CYAN}7. DNS 解析測試...${NC}"
if nslookup "$SIP_SERVER" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ DNS 解析正常${NC}"
else
    echo -e "${YELLOW}⚠ DNS 解析可能有問題${NC}"
fi

echo -e "\n${PURPLE}診斷完成${NC}"
echo -e "${YELLOW}如果發現問題，請根據上述提示進行修正${NC}"

# 提供問題解決建議
echo -e "\n${PURPLE}常見問題解決方案:${NC}"
echo -e "${CYAN}1. 交易超時 (transaction died):${NC}"
echo -e "   - 檢查 SIP 伺服器是否正常運作"
echo -e "   - 確認網路連通性"
echo -e "   - 檢查防火牆設定"
echo -e "   - 驗證 SIP 伺服器位址和端口"

echo -e "\n${CYAN}2. 網路設定檢查:${NC}"
echo -e "   - 確認客戶端 IP: $CLIENT_IP"
echo -e "   - 確認伺服器 IP: $SIP_SERVER"
echo -e "   - 確認端口: $SIP_PORT"

echo -e "\n${CYAN}3. 建議的測試步驟:${NC}"
echo -e "   - 先用 telnet/nc 測試基本連通性"
echo -e "   - 使用 tcpdump/wireshark 抓取網路封包"
echo -e "   - 檢查 SIP 伺服器日誌"
echo -e "   - 確認認證資訊正確" 