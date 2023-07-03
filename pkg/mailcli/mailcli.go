package mailcli

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
)

// NOTE: https://serverfault.com/questions/635139/how-to-fix-send-mail-authorization-failed-534-5-7-14
// MailClientOption represents a configuration option for the MailClient.
type MailClientOption func(*MailClient)

// MailClient represents a client for sending emails using Gmail.
type MailClient struct {
	senderEmail     string
	senderPassword  string
	template        string
	gmailServer     string
	gmailPort       int
	smtpServer      string
	smtpServerPort  int
	smtpServerAuth  smtp.Auth
	smtpServerPlain bool
}

// NewMailClient creates a new MailClient with the provided options.
func NewMailClient(ctx context.Context, options ...MailClientOption) (*MailClient, error) {
	client := &MailClient{
		senderEmail:     "your-email@gmail.com",
		senderPassword:  "default-password",
		template:        "Hello, %s!\n\nYour purchased password is: %s\n\nBest regards,\nThe Password Generator",
		gmailServer:     "smtp.gmail.com",
		gmailPort:       587,
		smtpServer:      "",
		smtpServerPort:  0,
		smtpServerAuth:  nil,
		smtpServerPlain: false,
	}

	// Apply options
	for _, opt := range options {
		opt(client)
	}

	// Validate the sender's email address
	if err := validateEmail(client.senderEmail); err != nil {
		return nil, err
	}

	// Configure SMTP server settings if not already configured
	if client.smtpServer == "" || client.smtpServerPort == 0 {
		if err := configureSMTPServer(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// configureSMTPServer sets up the SMTP server settings based on the Gmail server settings.
func configureSMTPServer(client *MailClient) error {
	auth := smtp.PlainAuth("", client.senderEmail, client.senderPassword, client.gmailServer)
	client.smtpServer = client.gmailServer
	client.smtpServerPort = client.gmailPort
	client.smtpServerAuth = auth
	client.smtpServerPlain = true

	return nil
}

// WithMailClientOptionSenderEmail sets the sender's email address.
func WithMailClientOptionSenderEmail(email string) MailClientOption {
	return func(c *MailClient) {
		c.senderEmail = email
	}
}

// WithMailClientOptionSenderPassword sets the sender's password address.
func WithMailClientOptionSenderPassword(password string) MailClientOption {
	return func(c *MailClient) {
		c.senderPassword = password
	}
}

// WithMailClientOptionTemplate sets the email template.
func WithMailClientOptionTemplate(template string) MailClientOption {
	return func(c *MailClient) {
		c.template = template
	}
}

// WithMailClientOptionGmailServer sets the Gmail server address and port.
func WithMailClientOptionGmailServer(server string, port int) MailClientOption {
	return func(c *MailClient) {
		c.gmailServer = server
		c.gmailPort = port
	}
}

// WithClientOptionSMTPServer sets the SMTP server address and port.
func WithClientOptionSMTPServer(server string, port int) MailClientOption {
	return func(c *MailClient) {
		c.smtpServer = server
		c.smtpServerPort = port
	}
}

// WithClientOptionSMTPServerAuth sets the SMTP server authentication method.
func WithClientOptionSMTPServerAuth(auth smtp.Auth) MailClientOption {
	return func(c *MailClient) {
		c.smtpServerAuth = auth
	}
}

// WithClientOptionSMTPServerPlain sets the SMTP server plain authentication method.
func WithClientOptionSMTPServerPlain(plain bool) MailClientOption {
	return func(c *MailClient) {
		c.smtpServerPlain = plain
	}
}

// SendPasswordEmail sends an email with an automatically generated password to the recipient.
func (c *MailClient) SendPasswordEmail(recipientEmail, password string) error {
	to := mail.Address{Name: "", Address: recipientEmail}
	from := mail.Address{Name: "", Address: c.senderEmail}
	subject := "New Password"

	body := fmt.Sprintf(c.template, to.Address, password)
	msg := []byte("To: " + to.String() + "\r\n" +
		"From: " + from.String() + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body)

	err := smtp.SendMail(fmt.Sprintf("%s:%d", c.smtpServer, c.smtpServerPort),
		c.smtpServerAuth, c.senderEmail, []string{recipientEmail}, msg)
	if err != nil {
		return err
	}

	return nil
}

// validateEmail checks if the given email address is a valid Gmail address.
func validateEmail(email string) error {
	if email == "" {
		return errors.New("sender's email address is required")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid sender's email address: %w", err)
	}

	at := strings.LastIndex(addr.Address, "@")
	var domain string
	if at < 0 {
		return errors.New("invalid email format")
	}

	domain = addr.Address[at+1:]
	if addr.Address != email || domain != "gmail.com" {
		return errors.New("only Gmail addresses are allowed")
	}

	return nil
}
