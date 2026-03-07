package mailer

import (
	"fmt"
	"log/slog"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

// Message represents an email message to send.
type Message struct {
	From    mail.Address
	To      []mail.Address
	Subject string
	HTML    string
}

// Service provides email sending and sender identity configuration.
type Service struct {
	SenderAddress string
	SenderName    string
	smtpHost      string
	smtpPort      int
	smtpUsername  string
	smtpPassword  string
	smtpTLS       bool
}

// New creates a mailer Service from environment variables.
// Required env vars: SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD,
// SENDER_ADDRESS, SENDER_NAME.
func New() *Service {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	return &Service{
		SenderAddress: os.Getenv("SENDER_ADDRESS"),
		SenderName:    os.Getenv("SENDER_NAME"),
		smtpHost:      os.Getenv("SMTP_HOST"),
		smtpPort:      port,
		smtpUsername:   os.Getenv("SMTP_USERNAME"),
		smtpPassword:   os.Getenv("SMTP_PASSWORD"),
		smtpTLS:        os.Getenv("SMTP_TLS") != "false",
	}
}

// Send sends an email message via SMTP. If SMTP is not configured, it
// logs the message at debug level and returns nil (best-effort).
func (s *Service) Send(msg *Message) error {
	if s.smtpHost == "" {
		slog.Debug("mailer: SMTP not configured, skipping email",
			"to", formatAddresses(msg.To),
			"subject", msg.Subject)
		return nil
	}

	from := msg.From
	if from.Address == "" {
		from = mail.Address{Address: s.SenderAddress, Name: s.SenderName}
	}

	toAddrs := make([]string, len(msg.To))
	for i, a := range msg.To {
		toAddrs[i] = a.Address
	}

	// Build MIME message
	var body strings.Builder
	body.WriteString("From: " + from.String() + "\r\n")
	body.WriteString("To: " + strings.Join(toAddrs, ", ") + "\r\n")
	body.WriteString("Subject: " + msg.Subject + "\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	body.WriteString("\r\n")
	body.WriteString(msg.HTML)

	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)
	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)

	if err := smtp.SendMail(addr, auth, from.Address, toAddrs, []byte(body.String())); err != nil {
		return fmt.Errorf("mailer: failed to send: %w", err)
	}
	return nil
}

// DefaultFrom returns a mail.Address with the configured sender identity.
func (s *Service) DefaultFrom() mail.Address {
	return mail.Address{Address: s.SenderAddress, Name: s.SenderName}
}

func formatAddresses(addrs []mail.Address) string {
	parts := make([]string, len(addrs))
	for i, a := range addrs {
		parts[i] = a.Address
	}
	return strings.Join(parts, ", ")
}
