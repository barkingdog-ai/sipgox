package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/barkingdog-ai/sipgox"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 設定日誌格式
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// 從環境變數讀取設定
	serverIP := os.Getenv("SIP_SERVER_IP")
	serverPort, _ := strconv.Atoi(os.Getenv("SIP_SERVER_PORT"))
	clientIP := os.Getenv("SIP_CLIENT_IP")
	clientPort, _ := strconv.Atoi(os.Getenv("SIP_CLIENT_PORT"))
	username := os.Getenv("SIP_USERNAME")
	password := os.Getenv("SIP_PASSWORD")

	// 檢查必要的環境變數
	if serverIP == "" || serverPort == 0 || clientIP == "" || clientPort == 0 || username == "" || password == "" {
		log.Fatal().Msg("請設定所有必要的環境變數 (SIP_SERVER_IP, SIP_SERVER_PORT, SIP_CLIENT_IP, SIP_CLIENT_PORT, SIP_USERNAME, SIP_PASSWORD)")
	}

	// 建立 SIP User Agent
	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent(username),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("建立 User Agent 失敗")
	}

	// 建立電話實例，設定監聽地址使用指定的客戶端 IP 和 Port
	phone := sipgox.NewPhone(ua,
		sipgox.WithPhoneLogger(log.Logger),
		sipgox.WithPhoneListenAddr(sipgox.ListenAddr{
			Network: "udp",
			Addr:    clientIP + ":" + strconv.Itoa(clientPort),
		}),
	)
	defer phone.Close()

	// 建立接聽的 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().
		Str("username", username).
		Str("server", serverIP).
		Int("port", serverPort).
		Str("client", clientIP).
		Int("client_port", clientPort).
		Msg("準備開始 SIP 服務")

	// 設定接聽選項，使用註冊地址自動處理註冊
	answerOpts := sipgox.AnswerOptions{
		Username:     username,
		Password:     password,
		RegisterAddr: serverIP + ":" + strconv.Itoa(serverPort), // 使用註冊地址自動處理註冊
		Expiry:       3600,                                      // 註冊過期時間
		Ringtime:     30 * time.Second,                          // 響鈴時間
		OnCall: func(inviteRequest *sip.Request) int {
			log.Info().
				Str("from", inviteRequest.From().String()).
				Str("to", inviteRequest.To().String()).
				Str("call_id", inviteRequest.CallID().Value()).
				Msg("收到來電")
			return 0 // 繼續處理來電 (接聽)
		},
	}

	log.Info().Msg("開始等待來電...")

	// 等待並接聽來電 (這會自動處理註冊)
	dialog, err := phone.Answer(ctx, answerOpts)
	if err != nil {
		log.Fatal().Err(err).Msg("接聽設定失敗")
	}

	if dialog != nil {
		log.Info().Msg("通話已建立")

		// 監聽對話狀態
		go func() {
			select {
			case <-dialog.Context().Done():
				log.Info().Msg("通話結束")
				cancel() // 通話結束時取消應用程式
			case <-ctx.Done():
				log.Info().Msg("應用程式即將結束，關閉通話")
				dialog.Close()
			}
		}()
	}

	// 等待中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Info().Msg("收到中斷信號，正在關閉...")
	case <-ctx.Done():
		log.Info().Msg("應用程式上下文已取消")
	}

	// 清理資源
	cancel()
	if dialog != nil {
		dialog.Close()
	}
	phone.Close()
	log.Info().Msg("程式結束")
}
