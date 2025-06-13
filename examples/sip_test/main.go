package main

import (
	"context"
	"os"
	"os/signal"
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

	// 建立 SIP User Agent
	ua, err := sipgo.NewUA()
	if err != nil {
		log.Fatal().Err(err).Msg("建立 User Agent 失敗")
	}

	// 建立電話實例
	phone := sipgox.NewPhone(ua, sipgox.WithPhoneLogger(log.Logger))

	// 設定 SIP 伺服器資訊
	serverURI := sip.Uri{
		Scheme: "sip",
		Host:   "192.168.11.210", // 請替換成你的 SIP 伺服器位址
		Port:   5060,
	}

	// 註冊到 SIP 伺服器
	ctx := context.Background()
	err = phone.Register(ctx, serverURI, sipgox.RegisterOptions{
		Username: "1001",     // 請替換成你的使用者名稱
		Password: "test1001", // 請替換成你的密碼
		Expiry:   3600,       // 註冊有效期（秒）
	})
	if err != nil {
		log.Fatal().Err(err).Msg("註冊失敗")
	}
	log.Info().Msg("成功註冊到 SIP 伺服器")

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
