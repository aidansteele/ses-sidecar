package sessidecar

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/emersion/go-smtp"
	"golang.org/x/exp/slog"
)

func Run(ctx context.Context, logger *slog.Logger, sesClient *ses.Client, addr string) error {
	bkd := &Backend{
		logger:  logger,
		ses:     sesClient,
		baseCtx: ctx,
	}

	s := smtp.NewServer(bkd)
	s.Addr = addr
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.AllowInsecureAuth = true

	logger.Info("Starting server", "addr", s.Addr)
	return s.ListenAndServe()
}

type Backend struct {
	logger  *slog.Logger
	ses     *ses.Client
	baseCtx context.Context
}

func (bkd *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	clientIP, clientPort, _ := net.SplitHostPort(conn.Conn().RemoteAddr().String())
	l := bkd.logger.With("clientIp", clientIP, "clientPort", clientPort)

	return &Session{
		logger:     l,
		baseLogger: l,
		ses:        bkd.ses,
		baseCtx:    bkd.baseCtx,
	}, nil
}

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

	messageID := *sent.MessageId
	l = l.With("sesMessageId", messageID)
	l.Info("Sent email")

	return &smtp.SMTPError{
		Code:         250,
		EnhancedCode: smtp.EnhancedCode{2, 0, 0},
		Message:      fmt.Sprintf("OK: queued as %s", messageID),
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
