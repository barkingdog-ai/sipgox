# SIP 撥號範例

這個範例展示如何使用 sipgox 套件進行 SIP 註冊和撥號。

## 功能特色

- **一站式註冊撥號**：使用 `sipgox.RegisterAndDial` 函數同時完成註冊和撥號
- **自動資源管理**：自動清理註冊和對話資源
- **動態被叫號碼**：支援從外部傳入被叫號碼
- **完整錯誤處理**：在各個階段都有適當的錯誤處理

## 使用方法

### 環境變數設定

設定以下環境變數：

```bash
export SIP_SERVER_IP="192.168.1.100"      # SIP 伺服器 IP
export SIP_SERVER_PORT="5060"             # SIP 伺服器端口
export SIP_CLIENT_IP="192.168.1.200"      # 客戶端 IP  
export SIP_CLIENT_PORT="5070"             # 客戶端端口
export SIP_CALLER_EXTENSION="301"         # 撥號方分機號碼
export SIP_CALLEE_EXTENSION="504"         # 被叫方分機號碼
export SIP_PASSWORD="yourpassword"        # SIP 密碼
```

### 執行範例

```bash
go run main.go
```

## 程式化使用

### 基本使用

```go
package main

import (
    "context"
    "log"
    
    "github.com/barkingdog-ai/sipgox"
)

func main() {
    // 設定參數
    params := sipgox.RegisterAndDialParams{
        ServerIP:        "192.168.1.100",
        ServerPort:      5060,
        ClientIP:        "192.168.1.200", 
        ClientPort:      5070,
        CallerExtension: "301",
        CalleeExtension: "504", // 可動態傳入被叫號碼
        Password:        "yourpassword",
        RegisterExpiry:  3600,  // 註冊過期時間（秒）
        DialTimeout:     60,    // 撥號超時時間（秒）
    }

    // 執行註冊和撥號
    result, err := sipgox.RegisterAndDial(context.Background(), params)
    if err != nil {
        log.Fatal("註冊並撥號失敗:", err)
    }

    // 確保清理資源
    defer result.Cancel()

    // 使用 result.Dialog 進行通話操作
    log.Println("通話已建立")
    
    // 等待通話結束
    <-result.Dialog.Context().Done()
    log.Println("通話結束")
}
```

### 進階使用

```go
// 動態撥號到不同號碼
func dialToExtension(extension string) error {
    params := sipgox.RegisterAndDialParams{
        // ... 其他參數
        CalleeExtension: extension, // 動態設定被叫號碼
    }
    
    result, err := sipgox.RegisterAndDial(context.Background(), params)
    if err != nil {
        return err
    }
    defer result.Cancel()
    
    // 處理通話邏輯
    return nil
}

// 批次撥號
func batchDial(extensions []string) {
    for _, ext := range extensions {
        go func(extension string) {
            if err := dialToExtension(extension); err != nil {
                log.Printf("撥號到 %s 失敗: %v", extension, err)
            }
        }(ext)
    }
}
```

## API 說明

### RegisterAndDialParams

| 欄位 | 類型 | 說明 |
|------|------|------|
| ServerIP | string | SIP 伺服器 IP 地址 |
| ServerPort | int | SIP 伺服器端口 |
| ClientIP | string | 客戶端 IP 地址 |
| ClientPort | int | 客戶端端口 |
| CallerExtension | string | 撥號方分機號碼 |
| CalleeExtension | string | 被叫方分機號碼 |
| Password | string | SIP 認證密碼 |
| RegisterExpiry | int | 註冊過期時間（秒），預設 3600 |
| DialTimeout | int | 撥號超時時間（秒），預設 60 |

### RegisterAndDialResult

| 欄位 | 類型 | 說明 |
|------|------|------|
| Dialog | *DialogClientSession | 撥號成功後的對話會話 |
| Phone | *Phone | 電話實例 |
| Cancel | context.CancelFunc | 清理函數，用於停止註冊和清理資源 |

## 注意事項

1. **資源清理**：務必呼叫 `result.Cancel()` 來清理資源
2. **網路設定**：確保客戶端 IP 和端口可以連接到 SIP 伺服器
3. **防火牆**：確保相關端口已開放
4. **並發使用**：每次撥號都會建立新的電話實例，支援併發使用 