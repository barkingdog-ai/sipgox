# SIP 客戶端範例

這個範例展示如何使用 sipgox 函式庫來建立一個 SIP 客戶端，該客戶端會自動註冊到 SIP 伺服器並等待來電。

## 特色

- 使用 `sipgox.Phone.Answer()` 的 `RegisterAddr` 選項自動處理 SIP 註冊
- 自動處理 SIP 認證
- 背景維持註冊狀態
- 接聽來電
- 優雅的關閉處理
- 避免端口衝突問題

## 使用方法

### 1. 設定環境變數

```bash
export SIP_SERVER_IP="你的SIP伺服器IP"
export SIP_SERVER_PORT="5060"
export SIP_CLIENT_IP="本機IP"
export SIP_CLIENT_PORT="5060"
export SIP_USERNAME="你的SIP用戶名"
export SIP_PASSWORD="你的SIP密碼"
```

### 2. 編譯程式

```bash
go build -o sip_client .
```

### 3. 執行程式

```bash
./sip_client
```

## 程式功能

1. **自動註冊**: 使用 `phone.Answer()` 的 `RegisterAddr` 選項自動註冊到 SIP 伺服器
2. **背景維持註冊**: 自動保持註冊狀態，無需手動管理
3. **接聽來電**: 等待並自動接聽來電
4. **日誌記錄**: 詳細的日誌記錄所有 SIP 操作
5. **優雅關閉**: 支援 SIGINT 和 SIGTERM 信號進行優雅關閉
6. **端口管理**: 避免端口衝突，統一使用 Answer 函數管理所有 SIP 操作

## 與原版差異

原版程式直接使用 `sipgo` 底層函式進行 SIP 操作，而這個版本使用 `sipgox` 專案提供的高階函式：

- 使用 `phone.Answer()` 的 `RegisterAddr` 選項統一處理註冊和接聽
- 移除手動註冊邏輯，避免端口衝突
- 自動處理認證和錯誤重試
- 更簡潔的程式碼結構
- 更穩定的資源管理

## 架構說明

程式採用單一進入點設計：

1. **建立 Phone 實例**: 配置監聽地址和日誌記錄器
2. **使用 Answer 方法**: 通過 `RegisterAddr` 選項自動處理註冊
3. **統一資源管理**: 所有 SIP 操作都通過 Answer 方法統一管理

這種設計避免了同時執行註冊和監聽導致的端口衝突問題。

## 注意事項

- 請確保防火牆設定允許 SIP 流量
- 確保 SIP 伺服器設定正確
- 建議在測試環境中先驗證設定
- 如果遇到端口已被占用的錯誤，請檢查是否有其他 SIP 程式正在運行 