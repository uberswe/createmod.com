package mailer

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

var headerReplacer = strings.NewReplacer("\r", "", "\n", "")

func sanitizeHeader(s string) string {
	return headerReplacer.Replace(s)
}

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
	body.WriteString("Subject: " + sanitizeHeader(msg.Subject) + "\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	body.WriteString("\r\n")
	body.WriteString(msg.HTML)

	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)
	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)
	tlsConfig := &tls.Config{ServerName: s.smtpHost}

	if s.smtpTLS && s.smtpPort == 465 {
		// Implicit TLS (port 465): connect with TLS from the start.
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("mailer: TLS dial failed: %w", err)
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, s.smtpHost)
		if err != nil {
			return fmt.Errorf("mailer: SMTP client failed: %w", err)
		}
		defer client.Close()
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer: auth failed: %w", err)
		}
		if err := client.Mail(from.Address); err != nil {
			return fmt.Errorf("mailer: MAIL FROM failed: %w", err)
		}
		for _, to := range toAddrs {
			if err := client.Rcpt(to); err != nil {
				return fmt.Errorf("mailer: RCPT TO failed: %w", err)
			}
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("mailer: DATA failed: %w", err)
		}
		if _, err := w.Write([]byte(body.String())); err != nil {
			return fmt.Errorf("mailer: write body failed: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("mailer: close body failed: %w", err)
		}
		return client.Quit()
	}

	// STARTTLS (port 587, 2525, etc.): connect plain, upgrade if available.
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("mailer: dial failed: %w", err)
	}
	defer client.Close()

	if s.smtpTLS {
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("mailer: STARTTLS failed: %w", err)
		}
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("mailer: auth failed: %w", err)
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("mailer: MAIL FROM failed: %w", err)
	}
	for _, to := range toAddrs {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("mailer: RCPT TO failed: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA failed: %w", err)
	}
	if _, err := w.Write([]byte(body.String())); err != nil {
		return fmt.Errorf("mailer: write body failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: close body failed: %w", err)
	}
	return client.Quit()
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
