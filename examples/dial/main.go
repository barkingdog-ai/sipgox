package main

import (
	"context"
	"fmt"
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

	log.Info().
		Str("caller", callerExtension).
		Str("callee", calleeExtension).
		Str("server", fmt.Sprintf("%s:%d", serverIP, serverPort)).
		Str("client", fmt.Sprintf("%s:%d", clientIP, clientPort)).
		Msg("開始撥號程式")

	// 建立 SIP User Agent - 設定正確的 hostname
	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent(callerExtension),
		// 關鍵修正：設定正確的 hostname，讓 From 域名與 PBX 一致
		sipgo.WithUserAgentHostname(serverIP),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("建立 User Agent 失敗")
	}

	// 建立電話實例
	phone := sipgox.NewPhone(ua,
		sipgox.WithPhoneLogger(log.Logger),
		sipgox.WithPhoneListenAddr(sipgox.ListenAddr{
			Network: "udp",
			Addr:    fmt.Sprintf("%s:%d", clientIP, clientPort),
		}),
	)
	defer phone.Close()

	// 建立主要的 context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// === 第一階段：註冊並保持狀態 ===
	log.Info().Str("username", callerExtension).Msg("開始註冊到 SIP 伺服器...")

	// 在背景啟動註冊，保持註冊狀態
	registerCtx, cancelRegister := context.WithCancel(ctx)
	defer cancelRegister() // 程式結束時取消註冊

	registerDone := make(chan error, 1)
	go func() {
		registerURI := sip.Uri{
			Host:    serverIP,
			Port:    serverPort,
			Headers: sip.HeaderParams{"transport": "udp"},
		}
		registerOpts := sipgox.RegisterOptions{
			Username: callerExtension,
			Password: password,
			Expiry:   3600,
		}

		// 這會進入 QualifyLoop 保持註冊狀態
		err := phone.Register(registerCtx, registerURI, registerOpts)
		registerDone <- err
	}()

	// 等待初始註冊完成（最多5秒）
	select {
	case err := <-registerDone:
		if err != nil {
			log.Warn().Err(err).Msg("註冊失敗，但繼續嘗試撥號")
		}
	case <-time.After(5 * time.Second):
		log.Info().Msg("註冊程序已在背景啟動，現在開始撥號")
	}

	log.Info().Msg("現在開始撥號...")

	// === 第二階段：撥號 ===
	recipient := sip.Uri{
		User:    calleeExtension,
		Host:    serverIP,
		Port:    serverPort,
		Headers: sip.HeaderParams{"transport": "udp"},
	}

	dialog, err := phone.Dial(ctx, recipient, sipgox.DialOptions{
		Username: callerExtension,
		Password: password,
		OnResponse: func(resp *sip.Response) {
			log.Info().
				Int("status", int(resp.StatusCode)).
				Str("reason", resp.Reason).
				Msg("收到回應")
		},
	})

	if err != nil {
		log.Fatal().Err(err).Msg("撥號失敗")
	}

	if dialog == nil {
		log.Fatal().Msg("撥號成功但沒有建立對話")
	}

	log.Info().Msg("撥號成功！通話已建立")

	// 監聽對話狀態
	go func() {
		<-dialog.Context().Done()
		log.Info().Msg("通話結束")
		cancel()
	}()

	// 等待通話結束或用戶中斷
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Info().Msg("收到中斷信號，正在掛斷通話...")
	case <-ctx.Done():
		log.Info().Msg("通話上下文已結束")
	}

	// 清理資源
	if dialog != nil {
		dialog.Close()
	}
	phone.Close()
	log.Info().Msg("程式結束")
}

// registerAndVerify 執行單次註冊確認（只等待 200 OK，不維持註冊狀態）
func registerAndVerify(ctx context.Context, phone *sipgox.Phone, serverIP string, serverPort int, clientIP string, clientPort int, username, password string) error {
	log.Info().
		Str("username", username).
		Str("server", fmt.Sprintf("%s:%d", serverIP, serverPort)).
		Msg("開始註冊確認到 SIP 伺服器")

	// 建立註冊目標 URI
	registerURI := sip.Uri{
		Host:    serverIP,
		Port:    serverPort,
		Headers: sip.HeaderParams{"transport": "udp"},
	}

	// 建立註冊選項
	registerOpts := sipgox.RegisterOptions{
		Username: username,
		Password: password,
		Expiry:   3600, // 1小時
	}

	// 設定10秒超時進行註冊確認
	registerCtx, cancelRegister := context.WithTimeout(ctx, 10*time.Second)
	defer cancelRegister()

	// 建立客戶端來執行單次註冊
	network := "udp"
	lhost := clientIP
	lport := clientPort

	client, err := sipgo.NewClient(phone.UA,
		sipgo.WithClientHostname(lhost),
		sipgo.WithClientPort(lport),
		sipgo.WithClientNAT(),
	)
	if err != nil {
		return fmt.Errorf("建立 SIP 客戶端失敗: %w", err)
	}
	defer client.Close()

	// 建立 Contact header
	contactHdr := sip.ContactHeader{
		Address: sip.Uri{
			User:    username,
			Host:    lhost,
			Port:    lport,
			Headers: sip.HeaderParams{"transport": network},
		},
	}

	// 建立註冊交易並執行單次註冊
	regTx := sipgox.NewRegisterTransaction(
		log.With().Str("component", "register").Logger(),
		client,
		registerURI,
		contactHdr,
		registerOpts,
	)

	// 執行註冊並等待 200 OK（不進入 QualifyLoop）
	if err := regTx.Register(registerCtx); err != nil {
		return fmt.Errorf("註冊確認失敗: %w", err)
	}

	log.Info().Msg("註冊確認成功！收到 200 OK，保持註冊狀態進行撥號")

	// 不立即取消註冊，讓註冊保持活躍狀態
	// 在主程式結束時會自動清理

	return nil
}
