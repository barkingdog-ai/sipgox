# 撥號腳本 (Dial Script)

這個腳本用於讓301分機撥打給504分機。

## 使用方法

1. **設定環境變數**：
   ```bash
   # 在examples/dial目錄下
   source .envrc
   ```

2. **執行撥號腳本**：
   ```bash
   go run main.go
   ```

## 設定說明

在 `.envrc` 檔案中包含以下設定：

- `SIP_SERVER_IP`: SIP伺服器IP地址
- `SIP_SERVER_PORT`: SIP伺服器埠號
- `SIP_CLIENT_IP`: 客戶端IP地址  
- `SIP_CLIENT_PORT`: 客戶端埠號
- `SIP_USERNAME`: 撥號者分機號碼 (301)
- `SIP_PASSWORD`: 撥號者密碼

## 腳本功能

- 使用301分機身份撥打給504分機
- 自動處理SIP認證
- 顯示撥號過程和狀態
- 支援手動掛斷 (Ctrl+C)
- 自動清理資源

## 撥號流程

1. 讀取環境變數設定
2. 建立SIP User Agent (身份：301)
3. 建立電話實例
4. 撥打504分機
5. 等待接通
6. 管理通話狀態
7. 處理掛斷和清理

## 測試

確保SIP伺服器上有301和504這兩個分機，且都已正確設定。

## 故障排除

- 檢查網路連接
- 確認SIP伺服器設定
- 驗證分機號碼和密碼
- 檢查防火牆設定 