package utils

import (
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPConfig holds connection settings for sending mail via SMTP (typically port 587 + STARTTLS).
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the RFC5322 From header value (e.g. "App Name <noreply@example.com>" or "a@b.com"). Empty uses Username.
	From string
}

const defaultSMTPPort = 587

func (c SMTPConfig) addr() string {
	port := c.Port
	if port == 0 {
		port = defaultSMTPPort
	}
	return fmt.Sprintf("%s:%d", c.Host, port)
}

func fromEnvelopeAddress(from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return from
	}
	if i := strings.LastIndex(from, "<"); i >= 0 && strings.HasSuffix(from, ">") {
		return strings.TrimSpace(from[i+1 : len(from)-1])
	}
	return from
}

// buildPlainTextMessage returns an RFC 5322 message with CRLF line endings.
func buildPlainTextMessage(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\nTo: ")
	b.WriteString(strings.Join(to, ", "))
	b.WriteString("\r\nSubject: ")
	b.WriteString(subject)
	b.WriteString("\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return []byte(b.String())
}

// SendEmailSMTP sends a plain-text email to the given recipients using SMTP AUTH (PLAIN) and STARTTLS when supported.
// Reuse this for any transactional email; build subject/body at the call site or in small wrappers (e.g. SendOTPEmail).
func SendEmailSMTP(cfg SMTPConfig, to []string, subject, body string) error {
	if cfg.Host == "" {
		return fmt.Errorf("smtp: host is not configured")
	}
	if len(to) == 0 {
		return fmt.Errorf("smtp: no recipients")
	}
	for _, addr := range to {
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("smtp: empty recipient address")
		}
	}

	from := strings.TrimSpace(cfg.From)
	if from == "" {
		from = cfg.Username
	}
	if from == "" {
		return fmt.Errorf("smtp: From and Username are empty")
	}

	envelopeFrom := fromEnvelopeAddress(from)
	if envelopeFrom == "" {
		return fmt.Errorf("smtp: could not determine envelope sender address")
	}

	msg := buildPlainTextMessage(from, to, subject, body)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	return smtp.SendMail(cfg.addr(), auth, envelopeFrom, to, msg)
}

// SendOTPEmail emails a one-time sign-in code using SendEmailSMTP.
func SendOTPEmail(cfg SMTPConfig, toEmail, otp string) error {
	subject := "Your sign-in verification code"
	body := fmt.Sprintf("Your verification code is: %s\n\nThis code expires in 5 minutes. If you did not request it, you can ignore this email.", otp)
	return SendEmailSMTP(cfg, []string{toEmail}, subject, body)
}
