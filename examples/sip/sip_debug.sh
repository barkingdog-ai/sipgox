#!/bin/bash

# 簡化的 SIP 監控調試腳本

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

DEFAULT_PORT="5060"
DEFAULT_INTERFACE="any"
SHOW_RAW_CONTENT=false

# 顯示說明
show_help() {
    echo -e "${PURPLE}SIP 監控調試工具${NC}"
    echo -e "${PURPLE}===============${NC}\n"
    echo "用法: $0 [選項]"
    echo ""
    echo "選項:"
    echo "  -p, --port PORT     監控端口 (預設: $DEFAULT_PORT)"
    echo "  -i, --interface IF  網路介面 (預設: $DEFAULT_INTERFACE)"
    echo "  -r, --raw           顯示原始 SIP 訊息內容"
    echo "  -h, --help          顯示此說明"
    echo ""
    echo "範例:"
    echo "  $0                  # 基本監控"
    echo "  $0 -r               # 顯示原始 SIP 內容"
    echo "  $0 -p 5061 -r       # 監控 5061 端口並顯示原始內容"
}

# 解析命令列參數
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            DEFAULT_PORT="$2"
            shift 2
            ;;
        -i|--interface)
            DEFAULT_INTERFACE="$2"
            shift 2
            ;;
        -r|--raw)
            SHOW_RAW_CONTENT=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}未知選項: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 檢查權限
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}錯誤: 此腳本需要 root 權限${NC}"
    echo "請使用: sudo $0"
    exit 1
fi

echo -e "${PURPLE}SIP 監控調試工具${NC}"
echo -e "${PURPLE}===============${NC}\n"

# 檢查工具
echo -e "${CYAN}檢查工具可用性...${NC}"
if command -v tshark &> /dev/null; then
    echo -e "${GREEN}✓ tshark 可用${NC}"
    TSHARK_VERSION=$(tshark -v | head -1)
    echo -e "  版本: $TSHARK_VERSION"
else
    echo -e "${RED}✗ tshark 不可用${NC}"
fi

if command -v tcpdump &> /dev/null; then
    echo -e "${GREEN}✓ tcpdump 可用${NC}"
    TCPDUMP_VERSION=$(tcpdump --version 2>&1 | head -1)
    echo -e "  版本: $TCPDUMP_VERSION"
else
    echo -e "${RED}✗ tcpdump 不可用${NC}"
fi

echo ""

# 檢查網路介面
echo -e "${CYAN}檢查網路介面...${NC}"
ip addr show | grep -E "^[0-9]+:" | sed 's/://g' | awk '{print "  " $2}' | head -5

echo ""

# 檢查是否有程式在監聽 5060 端口
echo -e "${CYAN}檢查端口 5060 使用狀況...${NC}"
netstat -ulpn | grep ":5060 " || echo -e "${YELLOW}  沒有程式在監聽 5060 端口${NC}"

echo ""

# 使用改進的 tshark 命令進行詳細監控
echo -e "${CYAN}開始監控 SIP 流量 (詳細模式)...${NC}"
echo -e "${YELLOW}介面: $DEFAULT_INTERFACE, 端口: $DEFAULT_PORT${NC}"
echo -e "${YELLOW}按 Ctrl+C 停止監控${NC}\n"

# 使用 tshark 進行詳細的 SIP 封包分析
echo -e "${GREEN}執行詳細監控命令...${NC}\n"

# 初始化變數
packet_count=0
in_packet=false
current_packet=""
packet_timestamp=""
packet_src=""
packet_dst=""

# 使用簡化但有效的 tshark 監控方式
echo -e "${GREEN}正在啟動 tshark 監控...${NC}"

# 顯示基本封包資訊的函數
show_packet_info() {
    local src_ip="$1"
    local dst_ip="$2" 
    local method="$3"
    local status="$4"
    local call_id="$5"
    local from_hdr="$6"
    local to_hdr="$7"
    local cseq="$8"
    
    echo -e "\n${PURPLE}╔══════════════════════════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                                    SIP 封包詳細資訊                                              ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════════════════════════╝${NC}"
    
    echo -e "${CYAN}📅 時間戳記:${NC} $(date '+%H:%M:%S')"
    echo -e "${CYAN}🌐 網路資訊:${NC}"
    echo -e "   └─ 來源: ${GREEN}$src_ip${NC}"
    echo -e "   └─ 目標: ${RED}$dst_ip${NC}"
    
    # SIP 方法或回應
    if [[ -n "$method" ]]; then
        echo -e "${YELLOW}📤 SIP 請求:${NC} ${BLUE}$method${NC}"
        case $method in
            "REGISTER")
                echo -e "   └─ ${GREEN}🔐 註冊請求 - 向 SIP 伺服器登記用戶${NC}"
                ;;
            "INVITE")
                echo -e "   └─ ${GREEN}📞 通話邀請 - 發起新的通話會話${NC}"
                ;;
            "ACK")
                echo -e "   └─ ${GREEN}✅ 確認應答 - 確認收到回應${NC}"
                ;;
            "BYE")
                echo -e "   └─ ${GREEN}👋 結束通話 - 終止通話會話${NC}"
                ;;
            "CANCEL")
                echo -e "   └─ ${GREEN}❌ 取消請求 - 取消進行中的請求${NC}"
                ;;
            "OPTIONS")
                echo -e "   └─ ${GREEN}❓ 選項查詢 - 查詢伺服器能力${NC}"
                ;;
        esac
    elif [[ -n "$status" ]]; then
        echo -e "${YELLOW}📥 SIP 回應:${NC} ${BLUE}$status${NC}"
        case $status in
            *"100 Trying"*)
                echo -e "   └─ ${CYAN}⏳ 處理中 - 伺服器正在處理請求${NC}"
                ;;
            *"180 Ringing"*)
                echo -e "   └─ ${YELLOW}🔔 響鈴中 - 目標用戶設備響鈴${NC}"
                ;;
            *"200 OK"*)
                echo -e "   └─ ${GREEN}✅ 成功 - 請求已成功處理${NC}"
                ;;
            *"401 Unauthorized"*)
                echo -e "   └─ ${RED}🔒 未授權 - 需要認證資訊${NC}"
                ;;
            *"403 Forbidden"*)
                echo -e "   └─ ${RED}🚫 禁止 - 伺服器拒絕請求${NC}"
                ;;
            *"404 Not Found"*)
                echo -e "   └─ ${RED}❓ 找不到 - 目標用戶不存在${NC}"
                ;;
            *"486 Busy Here"*)
                echo -e "   └─ ${YELLOW}📞 忙線中 - 目標用戶忙線${NC}"
                ;;
        esac
    fi
    
    # SIP 標頭詳細資訊
    echo -e "${CYAN}📋 SIP 標頭資訊:${NC}"
    
    if [[ -n "$call_id" ]]; then
        short_call_id=$(echo "$call_id" | cut -c1-20)
        echo -e "   🆔 Call-ID: ${GREEN}${short_call_id}...${NC}"
    fi
    
    if [[ -n "$cseq" ]]; then
        echo -e "   🔢 CSeq: ${GREEN}$cseq${NC}"
    fi
    
    if [[ -n "$from_hdr" ]]; then
        clean_from=$(echo "$from_hdr" | sed 's/[<>]//g' | cut -d';' -f1)
        echo -e "   👤 From: ${GREEN}$clean_from${NC}"
    fi
    
    if [[ -n "$to_hdr" ]]; then
        clean_to=$(echo "$to_hdr" | sed 's/[<>]//g' | cut -d';' -f1)
        echo -e "   👥 To: ${GREEN}$clean_to${NC}"
    fi
    
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                                      封包結束                                                   ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════════════════════════╝${NC}\n"
}

# 使用簡化但有效的 tshark 命令
tshark -i "$DEFAULT_INTERFACE" -f "udp port $DEFAULT_PORT" -Y "sip" -T fields \
       -e ip.src -e ip.dst -e sip.Method -e sip.Status-Line -e sip.Call-ID \
       -e sip.From -e sip.To -e sip.CSeq 2>/dev/null | \
while read -r src_ip dst_ip method status call_id from_hdr to_hdr cseq; do
    if [[ -n "$src_ip" && -n "$dst_ip" ]]; then
        show_packet_info "$src_ip" "$dst_ip" "$method" "$status" "$call_id" "$from_hdr" "$to_hdr" "$cseq"
    fi
done 