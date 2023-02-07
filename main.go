package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/emersion/go-smtp"
	"golang.org/x/exp/slog"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout))
	slog.SetDefault(logger)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	bkd := &Backend{
		logger:  logger,
		ses:     ses.NewFromConfig(cfg),
		baseCtx: ctx,
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:1025"
	}

	s := smtp.NewServer(bkd)
	s.Addr = addr
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.AllowInsecureAuth = true

	logger.Info("Starting server", "addr", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		logger.Error("starting server", err)
		os.Exit(1)
	}
}

type Backend struct {
	logger  *slog.Logger
	ses     *ses.Client
	baseCtx context.Context
}

func (bkd *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	clientIp, clientPort, _ := net.SplitHostPort(conn.Conn().RemoteAddr().String())
	l := bkd.logger.With("clientIp", clientIp, "clientPort", clientPort)

	return &Session{
		logger:     l,
		baseLogger: l,
		ses:        bkd.ses,
		baseCtx:    bkd.baseCtx,
	}, nil
}

// A Session is returned after EHLO.
type Session struct {
	logger     *slog.Logger
	baseLogger *slog.Logger
	ses        *ses.Client
	baseCtx    context.Context

	from       string
	recipients []string
}

func (s *Session) AuthPlain(username, password string) error {
	s.logger = s.logger.With("clientUsername", username)
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.logger.Debug("MAIL FROM", "from", from)
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string) error {
	s.recipients = append(s.recipients, to)
	s.logger.Debug("RCPT TO", "to", to, "recipients", s.recipients)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	ctx := s.baseCtx
	l := s.logger.With("from", s.from, "recipients", s.recipients)

	msg, err := io.ReadAll(r)
	if err != nil {
		l.Error("reading msg", err)
		return err
	}

	sent, err := s.ses.SendRawEmail(ctx, &ses.SendRawEmailInput{
		Source:       &s.from,
		Destinations: s.recipients,
		RawMessage:   &types.RawMessage{Data: msg},
	})
	if err != nil {
		l.Error("calling SendRawEmail", err)
		return err
	}

	messageId := *sent.MessageId
	l = l.With("sesMessageId", messageId)
	l.Info("Sent email")

	return &smtp.SMTPError{
		Code:         250,
		EnhancedCode: smtp.EnhancedCode{2, 0, 0},
		Message:      fmt.Sprintf("OK: queued as %s", messageId),
	}
}

func (s *Session) Reset() {
	s.logger = s.baseLogger
	s.from = ""
	s.recipients = nil
}

func (s *Session) Logout() error {
	return nil
}
