package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/barkingdog-ai/sipgox"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 設定日誌格式
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// 從環境變數讀取設定
	serverIP := os.Getenv("SIP_SERVER_IP")
	serverPortStr := os.Getenv("SIP_SERVER_PORT")
	clientIP := os.Getenv("SIP_CLIENT_IP")
	clientPortStr := os.Getenv("SIP_CLIENT_PORT")
	callerExtension := os.Getenv("SIP_CALLER_EXTENSION") // 撥號方分機 (301)
	calleeExtension := os.Getenv("SIP_CALLEE_EXTENSION") // 接聽方分機 (504)
	password := os.Getenv("SIP_PASSWORD")

	// 檢查必要的環境變數
	if serverIP == "" || serverPortStr == "" || clientIP == "" || clientPortStr == "" ||
		callerExtension == "" || calleeExtension == "" || password == "" {
		log.Fatal().Msg("請設定所有必要的環境變數 (SIP_SERVER_IP, SIP_SERVER_PORT, SIP_CLIENT_IP, SIP_CLIENT_PORT, SIP_CALLER_EXTENSION, SIP_CALLEE_EXTENSION, SIP_PASSWORD)")
	}

	serverPort, err := strconv.Atoi(serverPortStr)
	if err != nil {
		log.Fatal().Err(err).Msg("SIP_SERVER_PORT 必須是數字")
	}

	clientPort, err := strconv.Atoi(clientPortStr)
	if err != nil {
		log.Fatal().Err(err).Msg("SIP_CLIENT_PORT 必須是數字")
	}

	// 使用 sipgox.RegisterAndDial 函數
	ctx := context.Background()
	params := sipgox.RegisterAndDialParams{
		ServerIP:        serverIP,
		ServerPort:      serverPort,
		ClientIP:        clientIP,
		ClientPort:      clientPort,
		CallerExtension: callerExtension,
		CalleeExtension: calleeExtension,
		Password:        password,
		RegisterExpiry:  3600,
		DialTimeout:     60,
	}

	result, err := sipgox.RegisterAndDial(ctx, params)
	if err != nil {
		log.Fatal().Err(err).Msg("註冊並撥號失敗")
	}

	// 確保程式結束時清理資源
	defer result.Cancel()

	log.Info().Msg("通話已建立，按 Ctrl+C 結束通話")

	// 等待用戶中斷或通話結束
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Info().Msg("收到中斷信號，正在掛斷通話...")
	case <-result.Dialog.Context().Done():
		log.Info().Msg("通話已結束")
	}

	log.Info().Msg("程式結束")
}
