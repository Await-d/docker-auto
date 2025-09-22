package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/sirupsen/logrus"
)

// SESProvider implements EmailProvider for AWS SES
type SESProvider struct {
	config SESConfig
	client *ses.SES
	logger *logrus.Logger
}

// SESConfig holds AWS SES configuration
type SESConfig struct {
	Region      string
	AccessKey   string
	SecretKey   string
	From        string
	ConfigSet   string // Optional: for tracking

	// Advanced settings
	ReturnPath      string
	ReplyToAddress  string
	CharSet         string
}

// NewSESProvider creates a new AWS SES email provider
func NewSESProvider(config SESConfig, logger *logrus.Logger) (*SESProvider, error) {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Set defaults
	if config.CharSet == "" {
		config.CharSet = "UTF-8"
	}

	// Create AWS session
	awsConfig := &aws.Config{
		Region: aws.String(config.Region),
	}

	// Use explicit credentials if provided
	if config.AccessKey != "" && config.SecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			config.AccessKey,
			config.SecretKey,
			"", // token
		)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	client := ses.New(sess)

	return &SESProvider{
		config: config,
		client: client,
		logger: logger,
	}, nil
}

// Send sends an email via AWS SES
func (p *SESProvider) Send(ctx context.Context, message *Message) error {
	// Prepare the email input
	input := &ses.SendEmailInput{
		Source: aws.String(p.getFromAddress(message)),
		Destination: &ses.Destination{
			ToAddresses:  aws.StringSlice(message.To),
			CcAddresses:  aws.StringSlice(message.CC),
			BccAddresses: aws.StringSlice(message.BCC),
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Data:    aws.String(message.Subject),
				Charset: aws.String(p.config.CharSet),
			},
		},
	}

	// Set message body
	if message.HTMLBody != "" && message.TextBody != "" {
		// Both HTML and text
		input.Message.Body = &ses.Body{
			Html: &ses.Content{
				Data:    aws.String(message.HTMLBody),
				Charset: aws.String(p.config.CharSet),
			},
			Text: &ses.Content{
				Data:    aws.String(message.TextBody),
				Charset: aws.String(p.config.CharSet),
			},
		}
	} else if message.HTMLBody != "" {
		// HTML only
		input.Message.Body = &ses.Body{
			Html: &ses.Content{
				Data:    aws.String(message.HTMLBody),
				Charset: aws.String(p.config.CharSet),
			},
		}
	} else {
		// Text only
		input.Message.Body = &ses.Body{
			Text: &ses.Content{
				Data:    aws.String(message.TextBody),
				Charset: aws.String(p.config.CharSet),
			},
		}
	}

	// Optional settings
	if message.ReplyTo != "" {
		input.ReplyToAddresses = aws.StringSlice([]string{message.ReplyTo})
	} else if p.config.ReplyToAddress != "" {
		input.ReplyToAddresses = aws.StringSlice([]string{p.config.ReplyToAddress})
	}

	if p.config.ReturnPath != "" {
		input.ReturnPath = aws.String(p.config.ReturnPath)
	}

	if p.config.ConfigSet != "" {
		input.ConfigurationSetName = aws.String(p.config.ConfigSet)
	}

	// Add custom headers as tags (SES limitation: headers via raw email only)
	if len(message.Headers) > 0 {
		var tags []*ses.MessageTag
		for key, value := range message.Headers {
			// SES tags have restrictions, so we prefix with custom
			tagName := fmt.Sprintf("custom_%s", key)
			if len(tagName) > 256 {
				tagName = tagName[:256]
			}
			if len(value) > 256 {
				value = value[:256]
			}
			tags = append(tags, &ses.MessageTag{
				Name:  aws.String(tagName),
				Value: aws.String(value),
			})
		}
		input.Tags = tags
	}

	// Send the email
	result, err := p.client.SendEmailWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"to":         message.To,
		"subject":    message.Subject,
		"message_id": aws.StringValue(result.MessageId),
		"from":       p.getFromAddress(message),
	}).Info("Email sent successfully via AWS SES")

	return nil
}

// SendRaw sends a raw email via AWS SES (for advanced use cases)
func (p *SESProvider) SendRaw(ctx context.Context, rawMessage []byte, destinations []string) error {
	input := &ses.SendRawEmailInput{
		RawMessage: &ses.RawMessage{
			Data: rawMessage,
		},
		Destinations: aws.StringSlice(destinations),
	}

	if p.config.ConfigSet != "" {
		input.ConfigurationSetName = aws.String(p.config.ConfigSet)
	}

	result, err := p.client.SendRawEmailWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send raw email via SES: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"destinations": destinations,
		"message_id":   aws.StringValue(result.MessageId),
	}).Info("Raw email sent successfully via AWS SES")

	return nil
}

// ValidateConfig validates the SES configuration
func (p *SESProvider) ValidateConfig() error {
	if p.config.Region == "" {
		return fmt.Errorf("AWS region is required")
	}

	if p.config.From == "" {
		return fmt.Errorf("sender email address is required")
	}

	// Test if we can access SES
	_, err := p.client.GetSendQuota(&ses.GetSendQuotaInput{})
	if err != nil {
		return fmt.Errorf("failed to validate SES access: %w", err)
	}

	return nil
}

// GetProviderName returns the provider name
func (p *SESProvider) GetProviderName() string {
	return "ses"
}

// getFromAddress returns the from address for the message
func (p *SESProvider) getFromAddress(message *Message) string {
	if message.From != "" {
		return message.From
	}
	return p.config.From
}

// GetSendQuota returns the current send quota
func (p *SESProvider) GetSendQuota(ctx context.Context) (*SESQuota, error) {
	result, err := p.client.GetSendQuotaWithContext(ctx, &ses.GetSendQuotaInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get send quota: %w", err)
	}

	return &SESQuota{
		Max24HourSend:   aws.Float64Value(result.Max24HourSend),
		MaxSendRate:     aws.Float64Value(result.MaxSendRate),
		SentLast24Hours: aws.Float64Value(result.SentLast24Hours),
	}, nil
}

// GetSendStatistics returns send statistics
func (p *SESProvider) GetSendStatistics(ctx context.Context) ([]*SESStatistics, error) {
	result, err := p.client.GetSendStatisticsWithContext(ctx, &ses.GetSendStatisticsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get send statistics: %w", err)
	}

	var stats []*SESStatistics
	for _, data := range result.SendDataPoints {
		stats = append(stats, &SESStatistics{
			Timestamp:    aws.TimeValue(data.Timestamp),
			DeliveryAttempts: aws.Int64Value(data.DeliveryAttempts),
			Bounces:      aws.Int64Value(data.Bounces),
			Complaints:   aws.Int64Value(data.Complaints),
			Rejects:      aws.Int64Value(data.Rejects),
		})
	}

	return stats, nil
}

// VerifyEmailAddress verifies an email address with SES
func (p *SESProvider) VerifyEmailAddress(ctx context.Context, emailAddress string) error {
	input := &ses.VerifyEmailIdentityInput{
		EmailAddress: aws.String(emailAddress),
	}

	_, err := p.client.VerifyEmailIdentityWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to verify email address: %w", err)
	}

	return nil
}

// ListVerifiedEmailAddresses returns list of verified email addresses
func (p *SESProvider) ListVerifiedEmailAddresses(ctx context.Context) ([]string, error) {
	result, err := p.client.ListVerifiedEmailAddressesWithContext(ctx, &ses.ListVerifiedEmailAddressesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list verified email addresses: %w", err)
	}

	return aws.StringValueSlice(result.VerifiedEmailAddresses), nil
}

// CheckReputationStatus checks the reputation status (not available in current SES API)
func (p *SESProvider) CheckReputationStatus(ctx context.Context) (*SESReputation, error) {
	// Note: GetReputation is not available in the current AWS SES Go SDK
	// This is a placeholder that could be implemented with custom metrics
	return &SESReputation{
		BounceRate:    0.0,
		ComplaintRate: 0.0,
		DeliveryDelay: false,
	}, nil
}

// SESQuota represents SES sending quota
type SESQuota struct {
	Max24HourSend   float64 `json:"max_24_hour_send"`
	MaxSendRate     float64 `json:"max_send_rate"`
	SentLast24Hours float64 `json:"sent_last_24_hours"`
}

// SESStatistics represents SES send statistics
type SESStatistics struct {
	Timestamp        time.Time `json:"timestamp"`
	DeliveryAttempts int64     `json:"delivery_attempts"`
	Bounces          int64     `json:"bounces"`
	Complaints       int64     `json:"complaints"`
	Rejects          int64     `json:"rejects"`
}

// SESReputation represents SES reputation metrics
type SESReputation struct {
	BounceRate    float64 `json:"bounce_rate"`
	ComplaintRate float64 `json:"complaint_rate"`
	DeliveryDelay bool    `json:"delivery_delay"`
}