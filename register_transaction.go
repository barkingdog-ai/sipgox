package sipgox

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type RegisterTransaction struct {
	opts   RegisterOptions
	Origin *sip.Request

	client *sipgo.Client
	log    zerolog.Logger
}

func (t *RegisterTransaction) Terminate() error {
	return t.client.Close()
}

func NewRegisterTransaction(log zerolog.Logger, client *sipgo.Client, recipient sip.Uri, contact sip.ContactHeader, opts RegisterOptions) *RegisterTransaction {
	expiry, allowHDRS := opts.Expiry, opts.AllowHeaders
	// log := p.getLoggerCtx(ctx, "Register")
	req := sip.NewRequest(sip.REGISTER, recipient)
	req.AppendHeader(&contact)
	if expiry > 0 {
		expires := sip.ExpiresHeader(expiry)
		req.AppendHeader(&expires)
	}
	if allowHDRS != nil {
		req.AppendHeader(sip.NewHeader("Allow", strings.Join(allowHDRS, ", ")))
	}

	t := &RegisterTransaction{
		Origin: req, // origin maybe updated after first register
		opts:   opts,
		client: client,
		log:    log,
	}

	return t
}

func (p *RegisterTransaction) Register(ctx context.Context) error {
	username, password, expiry := p.opts.Username, p.opts.Password, p.opts.Expiry
	client := p.client
	log := p.log
	req := p.Origin
	contact := *req.Contact().Clone()

	// 為交易設定更長的超時時間，避免過早超時
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send request and parse response
	// req.SetDestination(*dst)
	log.Info().Str("uri", req.Recipient.String()).Int("expiry", int(expiry)).Msg("sending request")
	tx, err := client.TransactionRequest(ctxWithTimeout, req)
	if err != nil {
		return fmt.Errorf("fail to create transaction req=%q: %w", req.StartLine(), err)
	}
	defer tx.Terminate()

	res, err := getResponse(ctxWithTimeout, tx)
	if err != nil {
		// 提供更詳細的錯誤資訊
		if err.Error() == "transaction died" {
			return fmt.Errorf("SIP 伺服器沒有回應 REGISTER 請求，請檢查: 1) 伺服器是否運行在 %s 2) 網路連通性 3) 防火牆設定。原始錯誤: %w", req.Recipient.String(), err)
		}
		return fmt.Errorf("fail to get response req=%q : %w", req.StartLine(), err)
	}

	via := res.Via()
	if via == nil {
		return fmt.Errorf("no Via header in response")
	}

	// https://datatracker.ietf.org/doc/html/rfc3581#section-9
	if rport, _ := via.Params.Get("rport"); rport != "" {
		if p, err := strconv.Atoi(rport); err == nil {
			contact.Address.Port = p
		}

		if received, _ := via.Params.Get("received"); received != "" {
			// TODO: consider parsing IP
			contact.Address.Host = received
		}

		// Update contact address of NAT
		req.ReplaceHeader(&contact)
	}

	log.Info().Int("status", int(res.StatusCode)).Msg("Received status")
	if res.StatusCode == sip.StatusUnauthorized || res.StatusCode == sip.StatusProxyAuthRequired {
		tx.Terminate() //Terminate previous

		log.Info().Msg("Unauthorized. Doing digest auth")
		tx, err = client.DoDigestAuth(ctx, req, res, sipgo.DigestAuth{
			Username: username,
			Password: password,
		})
		if err != nil {
			return err
		}
		defer tx.Terminate()

		res, err = getResponse(ctx, tx)
		if err != nil {
			return fmt.Errorf("fail to get response req=%q : %w", req.StartLine(), err)
		}
		log.Info().Int("status", int(res.StatusCode)).Msg("Received status")
	}

	if res.StatusCode != 200 && res.StatusCode != 100 {
		return &RegisterResponseError{
			RegisterReq: req,
			RegisterRes: res,
			Msg:         res.StartLine(),
		}
	}

	return nil
}

func (t *RegisterTransaction) QualifyLoop(ctx context.Context) error {

	// TODO: based on server response Expires header this must be adjusted
	expiry := t.opts.Expiry
	if expiry == 0 {
		expiry = 30
	}

	// 在 expiry/2 時間重新註冊，確保註冊不會過期
	refreshInterval := time.Duration(expiry/2) * time.Second
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			t.log.Info().
				Int("expiry", expiry).
				Dur("refresh_interval", refreshInterval).
				Msg("執行定期註冊刷新")
		}
		err := t.qualify(ctx)
		if err != nil {
			t.log.Error().Err(err).Msg("註冊刷新失敗")
			return err
		}
	}
}

func (t *RegisterTransaction) Unregister(ctx context.Context) error {
	log := t.log
	req := t.Origin

	req.RemoveHeader("Expires")
	req.RemoveHeader("Contact")
	req.AppendHeader(sip.NewHeader("Contact", "*"))
	expires := sip.ExpiresHeader(0)
	req.AppendHeader(&expires)

	log.Info().Str("uri", req.Recipient.String()).Msg("UNREGISTER")
	return t.reregister(ctx, req)
}

func (t *RegisterTransaction) qualify(ctx context.Context) error {
	return t.reregister(ctx, t.Origin)
}

func (t *RegisterTransaction) reregister(ctx context.Context, req *sip.Request) error {
	log.Info().Msg("Reregistering")
	// log := p.getLoggerCtx(ctx, "Register")
	log := t.log
	client := t.client
	username, password := t.opts.Username, t.opts.Password
	// Send request and parse response
	// req.SetDestination(*dst)
	req.RemoveHeader("Via")
	tx, err := client.TransactionRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("fail to create transaction req=%q: %w", req.StartLine(), err)
	}
	defer tx.Terminate()

	res, err := getResponse(ctx, tx)
	if err != nil {
		return fmt.Errorf("fail to get response req=%q : %w", req.StartLine(), err)
	}

	log.Info().Int("status", int(res.StatusCode)).Msg("Received status")
	if res.StatusCode == sip.StatusUnauthorized || res.StatusCode == sip.StatusProxyAuthRequired {
		tx.Terminate() //Terminate previous
		log.Info().Msg("Unauthorized. Doing digest auth")
		tx, err = client.DoDigestAuth(ctx, req, res, sipgo.DigestAuth{
			Username: username,
			Password: password,
		})
		if err != nil {
			return err
		}
		defer tx.Terminate()

		res, err = getResponse(ctx, tx)
		if err != nil {
			return fmt.Errorf("fail to get response req=%q : %w", req.StartLine(), err)
		}
		log.Info().Int("status", int(res.StatusCode)).Msg("Received status")
	}

	if res.StatusCode != 200 && res.StatusCode != 100 {
		return &RegisterResponseError{
			RegisterReq: req,
			RegisterRes: res,
			Msg:         res.StartLine(),
		}
	}

	return nil
}
