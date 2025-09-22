package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SMTPProvider implements EmailProvider for SMTP
type SMTPProvider struct {
	config SMTPConfig
	logger *logrus.Logger
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string

	// TLS settings
	UseTLS       bool
	UseStartTLS  bool
	SkipVerifyTLS bool

	// Connection settings
	ConnectTimeout time.Duration
	SendTimeout    time.Duration

	// Pool settings
	MaxIdleConns   int
	MaxOpenConns   int
	ConnMaxLifetime time.Duration
}

// NewSMTPProvider creates a new SMTP email provider
func NewSMTPProvider(config SMTPConfig, logger *logrus.Logger) *SMTPProvider {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Set defaults
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 30 * time.Second
	}
	if config.SendTimeout == 0 {
		config.SendTimeout = 60 * time.Second
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 100
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = 1 * time.Hour
	}

	return &SMTPProvider{
		config: config,
		logger: logger,
	}
}

// Send sends an email via SMTP
func (p *SMTPProvider) Send(ctx context.Context, message *Message) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, p.config.SendTimeout)
	defer cancel()

	// Build the email
	emailData, err := p.buildEmail(message)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	// Establish connection
	client, err := p.createSMTPClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials provided
	if p.config.Username != "" && p.config.Password != "" {
		auth := smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	from := message.From
	if from == "" {
		from = p.config.From
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	allRecipients := append(message.To, message.CC...)
	allRecipients = append(allRecipients, message.BCC...)

	for _, recipient := range allRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
		}
	}

	// Send the email data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := writer.Write(emailData); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write email data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"to":      message.To,
		"subject": message.Subject,
		"from":    from,
	}).Info("Email sent successfully via SMTP")

	return nil
}

// ValidateConfig validates the SMTP configuration
func (p *SMTPProvider) ValidateConfig() error {
	if p.config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if p.config.Port <= 0 || p.config.Port > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", p.config.Port)
	}

	if p.config.From == "" {
		return fmt.Errorf("sender email address is required")
	}

	return nil
}

// GetProviderName returns the provider name
func (p *SMTPProvider) GetProviderName() string {
	return "smtp"
}

// createSMTPClient creates an SMTP client with proper TLS configuration
func (p *SMTPProvider) createSMTPClient(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(p.config.Host, strconv.Itoa(p.config.Port))

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: p.config.ConnectTimeout,
	}

	var conn net.Conn
	var err error

	// Create connection based on TLS configuration
	if p.config.UseTLS {
		// Direct TLS connection (usually port 465)
		tlsConfig := &tls.Config{
			ServerName:         p.config.Host,
			InsecureSkipVerify: p.config.SkipVerifyTLS,
		}

		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to establish TLS connection: %w", err)
		}
	} else {
		// Plain connection (usually port 587 with STARTTLS)
		conn, err = dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, fmt.Errorf("failed to establish connection: %w", err)
		}
	}

	// Create SMTP client
	client, err := smtp.NewClient(conn, p.config.Host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	// Use STARTTLS if not already using TLS
	if !p.config.UseTLS && p.config.UseStartTLS {
		tlsConfig := &tls.Config{
			ServerName:         p.config.Host,
			InsecureSkipVerify: p.config.SkipVerifyTLS,
		}

		if err := client.StartTLS(tlsConfig); err != nil {
			client.Close()
			return nil, fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	return client, nil
}

// buildEmail builds the raw email message
func (p *SMTPProvider) buildEmail(message *Message) ([]byte, error) {
	var builder strings.Builder

	// Headers
	from := message.From
	if from == "" {
		from = p.config.From
	}
	builder.WriteString(fmt.Sprintf("From: %s\r\n", from))

	if len(message.To) > 0 {
		builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(message.To, ", ")))
	}

	if len(message.CC) > 0 {
		builder.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(message.CC, ", ")))
	}

	if message.ReplyTo != "" {
		builder.WriteString(fmt.Sprintf("Reply-To: %s\r\n", message.ReplyTo))
	}

	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Subject))
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	builder.WriteString(fmt.Sprintf("Message-ID: <%s@%s>\r\n",
		fmt.Sprintf("%d", time.Now().UnixNano()), p.config.Host))

	// Custom headers
	for key, value := range message.Headers {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Priority
	switch message.Priority {
	case PriorityHigh:
		builder.WriteString("X-Priority: 1\r\n")
		builder.WriteString("X-MSMail-Priority: High\r\n")
	case PriorityUrgent:
		builder.WriteString("X-Priority: 1\r\n")
		builder.WriteString("X-MSMail-Priority: High\r\n")
		builder.WriteString("Importance: high\r\n")
	case PriorityLow:
		builder.WriteString("X-Priority: 5\r\n")
		builder.WriteString("X-MSMail-Priority: Low\r\n")
	}

	// Content type
	if message.HTMLBody != "" && message.TextBody != "" {
		// Multipart alternative
		boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		builder.WriteString("\r\n")

		// Text part
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		builder.WriteString(message.TextBody)
		builder.WriteString("\r\n\r\n")

		// HTML part
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		builder.WriteString(message.HTMLBody)
		builder.WriteString("\r\n\r\n")

		builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if message.HTMLBody != "" {
		// HTML only
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		builder.WriteString(message.HTMLBody)
	} else {
		// Text only
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		builder.WriteString(message.TextBody)
	}

	return []byte(builder.String()), nil
}

// TestConnection tests the SMTP connection
func (p *SMTPProvider) TestConnection(ctx context.Context) error {
	client, err := p.createSMTPClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Test authentication if credentials provided
	if p.config.Username != "" && p.config.Password != "" {
		auth := smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication test failed: %w", err)
		}
	}

	return nil
}