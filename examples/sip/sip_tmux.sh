#!/bin/bash

# SIP 監控和測試的 tmux 會話腳本
# 左邊: SIP 監控 (sip_debug.sh)
# 右邊: SIP 客戶端 (main.go)

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 設定
SESSION_NAME="sip-monitor"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 預設 SIP 設定 (使用實際網路環境)
DEFAULT_SIP_SERVER_IP="192.168.11.210"  # 假設的 SIP 伺服器
DEFAULT_SIP_SERVER_PORT="5060"
DEFAULT_SIP_CLIENT_IP="192.168.11.204"   # 您的實際 IP
DEFAULT_SIP_CLIENT_PORT="5061"
DEFAULT_SIP_USERNAME="testuser"
DEFAULT_SIP_PASSWORD="testpass"

# 顯示說明
show_help() {
    echo -e "${PURPLE}SIP 監控和測試 tmux 會話${NC}"
    echo -e "${PURPLE}========================${NC}\n"
    echo "此腳本會創建一個 tmux 會話，包含："
    echo "  左邊: SIP 流量監控 (sip_debug.sh)"
    echo "  右邊: SIP 客戶端程式 (main.go)"
    echo ""
    echo -e "${CYAN}用法:${NC} $0 [選項]"
    echo ""
    echo -e "${CYAN}選項:${NC}"
    echo "  -s, --server IP      SIP 伺服器 IP (預設: $DEFAULT_SIP_SERVER_IP)"
    echo "  -p, --port PORT      SIP 伺服器端口 (預設: $DEFAULT_SIP_SERVER_PORT)"
    echo "  -c, --client IP      SIP 客戶端 IP (預設: $DEFAULT_SIP_CLIENT_IP)"
    echo "  -C, --client-port P  SIP 客戶端端口 (預設: $DEFAULT_SIP_CLIENT_PORT)"
    echo "  -u, --username USER  SIP 用戶名 (預設: $DEFAULT_SIP_USERNAME)"
    echo "  -P, --password PASS  SIP 密碼 (預設: $DEFAULT_SIP_PASSWORD)"
    echo "  -r, --raw            監控時顯示原始 SIP 內容"
    echo "  -k, --kill           終止現有的 tmux 會話"
    echo "  -h, --help           顯示此說明"
    echo ""
    echo -e "${CYAN}範例:${NC}"
    echo "  $0                                    # 使用預設設定"
    echo "  $0 -s 10.0.0.1 -u myuser -P mypass   # 自訂伺服器和認證"
    echo "  $0 -r                                 # 顯示原始 SIP 內容"
    echo "  $0 -k                                 # 終止現有會話"
    echo ""
    echo -e "${YELLOW}注意:${NC}"
    echo "  • SIP 監控需要 root 權限"
    echo "  • 請確保已安裝 tmux 和 tshark"
    echo "  • 按 Ctrl+B 然後 D 來分離 tmux 會話"
    echo "  • 使用 'tmux attach -t $SESSION_NAME' 重新連接"
}

# 檢查依賴
check_dependencies() {
    local missing_tools=()
    
    if ! command -v tmux &> /dev/null; then
        missing_tools+=("tmux")
    fi
    
    if ! command -v tshark &> /dev/null; then
        missing_tools+=("tshark")
    fi
    
    if ! command -v go &> /dev/null; then
        missing_tools+=("go")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        echo -e "${RED}錯誤: 缺少必要工具: ${missing_tools[*]}${NC}"
        echo ""
        echo "安裝指令:"
        for tool in "${missing_tools[@]}"; do
            case $tool in
                "tmux")
                    echo "  sudo apt-get install tmux      # Ubuntu/Debian"
                    echo "  sudo yum install tmux          # CentOS/RHEL"
                    ;;
                "tshark")
                    echo "  sudo apt-get install tshark    # Ubuntu/Debian"
                    echo "  sudo yum install wireshark     # CentOS/RHEL"
                    ;;
                "go")
                    echo "  請從 https://golang.org/dl/ 安裝 Go"
                    ;;
            esac
        done
        exit 1
    fi
}

# 終止現有會話
kill_session() {
    if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        echo -e "${YELLOW}終止現有的 tmux 會話: $SESSION_NAME${NC}"
        tmux kill-session -t "$SESSION_NAME"
        echo -e "${GREEN}會話已終止${NC}"
    else
        echo -e "${YELLOW}沒有找到名為 '$SESSION_NAME' 的會話${NC}"
    fi
}

# 解析命令列參數
SIP_SERVER_IP="$DEFAULT_SIP_SERVER_IP"
SIP_SERVER_PORT="$DEFAULT_SIP_SERVER_PORT"
SIP_CLIENT_IP="$DEFAULT_SIP_CLIENT_IP"
SIP_CLIENT_PORT="$DEFAULT_SIP_CLIENT_PORT"
SIP_USERNAME="$DEFAULT_SIP_USERNAME"
SIP_PASSWORD="$DEFAULT_SIP_PASSWORD"
SHOW_RAW=false
KILL_SESSION=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--server)
            SIP_SERVER_IP="$2"
            shift 2
            ;;
        -p|--port)
            SIP_SERVER_PORT="$2"
            shift 2
            ;;
        -c|--client)
            SIP_CLIENT_IP="$2"
            shift 2
            ;;
        -C|--client-port)
            SIP_CLIENT_PORT="$2"
            shift 2
            ;;
        -u|--username)
            SIP_USERNAME="$2"
            shift 2
            ;;
        -P|--password)
            SIP_PASSWORD="$2"
            shift 2
            ;;
        -r|--raw)
            SHOW_RAW=true
            shift
            ;;
        -k|--kill)
            KILL_SESSION=true
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

# 如果要求終止會話
if [ "$KILL_SESSION" = true ]; then
    kill_session
    exit 0
fi

# 檢查依賴工具
check_dependencies

# 檢查檔案是否存在
if [ ! -f "$SCRIPT_DIR/sip_debug.sh" ]; then
    echo -e "${RED}錯誤: 找不到 sip_debug.sh${NC}"
    exit 1
fi

if [ ! -f "$SCRIPT_DIR/main.go" ]; then
    echo -e "${RED}錯誤: 找不到 main.go${NC}"
    exit 1
fi

# 終止現有會話（如果存在）
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    echo -e "${YELLOW}發現現有會話，正在終止...${NC}"
    tmux kill-session -t "$SESSION_NAME"
fi

echo -e "${PURPLE}建立 SIP 監控和測試會話${NC}"
echo -e "${PURPLE}========================${NC}\n"

echo -e "${CYAN}設定資訊:${NC}"
echo -e "  🌐 SIP 伺服器: ${GREEN}$SIP_SERVER_IP:$SIP_SERVER_PORT${NC}"
echo -e "  💻 SIP 客戶端: ${GREEN}$SIP_CLIENT_IP:$SIP_CLIENT_PORT${NC}"
echo -e "  👤 用戶名: ${GREEN}$SIP_USERNAME${NC}"
echo -e "  🔑 密碼: ${GREEN}$SIP_PASSWORD${NC}"
echo -e "  📊 顯示原始內容: ${GREEN}$([ $SHOW_RAW == true ] && echo '是' || echo '否')${NC}"
echo ""

# 建立新的 tmux 會話
echo -e "${GREEN}建立 tmux 會話: $SESSION_NAME${NC}"

# 建立會話並設定第一個窗格（左邊 - SIP 監控）
tmux new-session -d -s "$SESSION_NAME" -x 120 -y 40

# 設定窗格標題
tmux rename-window -t "$SESSION_NAME:0" "SIP-Monitor"

# 分割窗格為左右兩個
tmux split-window -h -t "$SESSION_NAME:0"

# 設定左邊窗格（SIP 監控）
monitor_cmd="sudo $SCRIPT_DIR/sip_debug.sh -p $SIP_SERVER_PORT"
if [ "$SHOW_RAW" = true ]; then
    monitor_cmd="$monitor_cmd -r"
fi

echo -e "${CYAN}左邊窗格: SIP 監控${NC}"
tmux send-keys -t "$SESSION_NAME:0.0" "echo '=== SIP 流量監控 ==='" Enter
tmux send-keys -t "$SESSION_NAME:0.0" "echo '監控端口: $SIP_SERVER_PORT'" Enter
tmux send-keys -t "$SESSION_NAME:0.0" "echo '按 Ctrl+C 停止監控'" Enter
tmux send-keys -t "$SESSION_NAME:0.0" "echo ''" Enter
tmux send-keys -t "$SESSION_NAME:0.0" "$monitor_cmd" Enter

# 設定右邊窗格（SIP 客戶端）
echo -e "${CYAN}右邊窗格: SIP 客戶端${NC}"
tmux send-keys -t "$SESSION_NAME:0.1" "cd $SCRIPT_DIR" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '=== SIP 客戶端程式 ==='" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '設定環境變數...'" Enter

# 設定環境變數
tmux send-keys -t "$SESSION_NAME:0.1" "export SIP_SERVER_IP='$SIP_SERVER_IP'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "export SIP_SERVER_PORT='$SIP_SERVER_PORT'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "export SIP_CLIENT_IP='$SIP_CLIENT_IP'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "export SIP_CLIENT_PORT='$SIP_CLIENT_PORT'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "export SIP_USERNAME='$SIP_USERNAME'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "export SIP_PASSWORD='$SIP_PASSWORD'" Enter

tmux send-keys -t "$SESSION_NAME:0.1" "echo '環境變數已設定:'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '  SIP_SERVER_IP=$SIP_SERVER_IP'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '  SIP_SERVER_PORT=$SIP_SERVER_PORT'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '  SIP_CLIENT_IP=$SIP_CLIENT_IP'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '  SIP_CLIENT_PORT=$SIP_CLIENT_PORT'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '  SIP_USERNAME=$SIP_USERNAME'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo ''" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '準備執行 SIP 客戶端...'" Enter
tmux send-keys -t "$SESSION_NAME:0.1" "echo '輸入 \"go run main.go\" 開始'" Enter

# 分割右邊窗格為上下兩個（上面是 main.go，下面是測試腳本）
tmux split-window -v -t "$SESSION_NAME:0.1"

# 設定右下窗格（測試腳本）
echo -e "${CYAN}右下窗格: SIP 測試工具${NC}"
tmux send-keys -t "$SESSION_NAME:0.2" "cd $SCRIPT_DIR" Enter
tmux send-keys -t "$SESSION_NAME:0.2" "echo '=== SIP 測試封包產生器 ==='" Enter
tmux send-keys -t "$SESSION_NAME:0.2" "echo '輸入以下命令發送測試封包:'" Enter
tmux send-keys -t "$SESSION_NAME:0.2" "echo '  ./test_sip_packets.sh'" Enter
tmux send-keys -t "$SESSION_NAME:0.2" "echo '  ./test_sip_packets.sh [目標IP] [端口]'" Enter
tmux send-keys -t "$SESSION_NAME:0.2" "echo ''" Enter
tmux send-keys -t "$SESSION_NAME:0.2" "echo '預設會發送到: $SIP_CLIENT_IP:$SIP_SERVER_PORT'" Enter

# 設定窗格大小
tmux resize-pane -t "$SESSION_NAME:0.0" -R 20  # 左邊監控窗格更寬
tmux resize-pane -t "$SESSION_NAME:0.1" -D 10  # 右上 main.go 窗格

# 連接到會話
echo -e "${GREEN}tmux 會話已建立！${NC}"
echo -e "${GREEN}佈局說明:${NC}"
echo -e "  📊 左邊: SIP 流量監控"
echo -e "  💻 右上: SIP 客戶端程式 (main.go)"  
echo -e "  🧪 右下: SIP 測試工具"
echo ""
echo -e "${YELLOW}使用方式:${NC}"
echo -e "  1. 監控已自動開始"
echo -e "  2. 在右下窗格執行: ./test_sip_packets.sh"
echo -e "  3. 在右上窗格執行: go run main.go"
echo -e "  4. 觀察左邊監控窗格的 SIP 流量"
echo ""
echo -e "${YELLOW}tmux 操作:${NC}"
echo -e "  • Ctrl+B 然後 D: 分離會話"
echo -e "  • Ctrl+B 然後 方向鍵: 切換窗格"
echo -e "  • tmux attach -t $SESSION_NAME: 重新連接"
echo -e "  • $0 -k: 終止會話"
echo ""

# 自動連接到會話
tmux attach-session -t "$SESSION_NAME" 