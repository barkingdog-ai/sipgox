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

	// 建立電話實例
	phone := sipgox.NewPhone(ua, sipgox.WithPhoneLogger(log.Logger))

	// 設定 SIP 伺服器資訊
	serverURI := sip.Uri{
		Scheme: "sip",
		Host:   serverIP,
		Port:   serverPort,
		User:   username,
	}

	// 註冊到 SIP 伺服器
	maxRetries := 2 // 減少重試次數
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second) // 減少超時時間

		log.Info().
			Int("attempt", i+1).
			Str("username", username).
			Str("server", serverURI.Host).
			Msg("嘗試註冊到 SIP 伺服器")

		// 建立 Client
		client, err := sipgo.NewClient(ua,
			sipgo.WithClientHostname(clientIP),
			sipgo.WithClientPort(clientPort),
			sipgo.WithClientNAT(),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("建立 Client 失敗")
		}
		defer client.Close()

		// 建立 Contact 標頭
		contactHdr := sip.ContactHeader{
			Address: sip.Uri{
				User:      username,
				Host:      clientIP,
				Port:      clientPort,
				Headers:   sip.HeaderParams{"transport": "udp"},
				UriParams: sip.NewParams(),
			},
			Params: sip.NewParams(),
		}

		// 建立註冊請求
		req := sip.NewRequest(sip.REGISTER, serverURI)
		req.AppendHeader(&contactHdr)
		expires := sip.ExpiresHeader(3600)
		req.AppendHeader(&expires)

		// 發送註冊請求
		tx, err := client.TransactionRequest(ctx, req)
		if err != nil {
			log.Error().Err(err).Msg("發送註冊請求失敗")
			continue
		}
		defer tx.Terminate()

		// 等待回應
		var res *sip.Response
		select {
		case <-tx.Done():
			log.Error().Msg("交易結束")
			continue
		case res = <-tx.Responses():
		case <-ctx.Done():
			log.Error().Msg("等待回應超時")
			continue
		}

		// 處理回應
		if res.StatusCode == sip.StatusUnauthorized || res.StatusCode == sip.StatusProxyAuthRequired {
			// 進行摘要認證
			tx.Terminate()
			tx, err = client.DoDigestAuth(ctx, req, res, sipgo.DigestAuth{
				Username: username,
				Password: password,
			})
			if err != nil {
				log.Error().Err(err).Msg("摘要認證失敗")
				continue
			}
			defer tx.Terminate()

			// 等待認證回應
			select {
			case <-tx.Done():
				log.Error().Msg("認證交易結束")
				continue
			case res = <-tx.Responses():
			case <-ctx.Done():
				log.Error().Msg("等待認證回應超時")
				continue
			}
		}

		// 處理各種狀態碼
		switch res.StatusCode {
		case sip.StatusTrying:
			log.Info().
				Int("status", int(res.StatusCode)).
				Str("reason", res.Reason).
				Msg("註冊請求正在處理中")
			continue
		case sip.StatusOK:
			log.Info().
				Str("username", username).
				Str("server", serverURI.Host).
				Int("status", int(res.StatusCode)).
				Str("reason", res.Reason).
				Str("contact", contactHdr.String()).
				Str("expires", expires.String()).
				Msg("成功註冊到 SIP 伺服器")
			return
		default:
			log.Error().
				Int("status", int(res.StatusCode)).
				Str("reason", res.Reason).
				Msg("註冊失敗")
			continue
		}
	}

	if lastErr != nil {
		log.Error().
			Err(lastErr).
			Str("username", username).
			Str("server", serverURI.Host).
			Msg("多次嘗試註冊都失敗")
		return
	}

	// 設定接聽選項
	answerOpts := sipgox.AnswerOptions{
		Expiry:   3600,
		Ringtime: 30 * time.Second,
		OnCall: func(inviteRequest *sip.Request) int {
			log.Info().Str("from", inviteRequest.From().String()).Msg("收到來電")
			return 0 // 繼續處理來電
		},
	}

	// 等待並接聽來電
	answerCtx := context.Background()
	log.Info().Msg("開始等待來電...")
	dialog, err := phone.Answer(answerCtx, answerOpts)
	if err != nil {
		log.Fatal().Err(err).Msg("接聽設定失敗")
	}

	// 監聽對話狀態
	go func() {
		select {
		case <-dialog.Context().Done():
			log.Info().Msg("通話結束")
			return
		}
	}()

	// 等待中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 清理資源
	phone.Close()
	log.Info().Msg("程式結束")
}
