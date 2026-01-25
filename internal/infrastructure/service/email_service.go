package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"net"
	"net/smtp"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	textTemplate "text/template"
	"time"

	"core-backend/config"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/pkg/utils"

	"go.uber.org/zap"
)

// emailService handles email sending via Gmail SMTP with rate limiting
type emailService struct {
	smtpHost      string
	smtpPort      int
	username      string
	password      string
	fromName      string
	fromEmail     string
	templates     *htmlTemplate.Template
	textTemplates *textTemplate.Template
	rateLimiter   *rateLimiter
	connPool      *smtpConnectionPool
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     time.Duration
	lastRefillTime time.Time
	mu             sync.Mutex
}

// smtpConnectionPool manages a pool of SMTP connections for reuse
type smtpConnectionPool struct {
	connections chan *smtpConnection
	maxSize     int
	smtpHost    string
	smtpPort    int
	username    string
	password    string
	mu          sync.Mutex
}

// smtpConnection wraps an SMTP client with metadata
type smtpConnection struct {
	client    *smtp.Client
	conn      net.Conn
	createdAt time.Time
	lastUsed  time.Time
}

// NewEmailService creates a new email service instance with rate limiting
func NewEmailService(cfg *config.AppConfig) (iservice_third_party.EmailService, error) {
	// Validate configuration
	if cfg.GmailSMTP.Host == "" {
		return nil, errors.New("gmail SMTP host is required")
	}
	if cfg.GmailSMTP.Username == "" {
		return nil, errors.New("gmail SMTP username is required")
	}
	if cfg.GmailSMTP.Password == "" {
		return nil, errors.New("gmail app password is required")
	}
	if cfg.GmailSMTP.FromEmail == "" {
		return nil, errors.New("from email address is required")
	}

	// Calculate refill rate based on rate limit (emails per minute)
	emailsPerMinute := cfg.Notification.RateLimits.EmailPerMinute
	if emailsPerMinute <= 0 {
		emailsPerMinute = 30 // Default to 30 per minute (Gmail limit)
	}
	refillRate := time.Minute / time.Duration(emailsPerMinute)

	// Initialize connection pool (size = number of concurrent workers)
	poolSize := cfg.Notification.ConsumerConcurrency.Email
	if poolSize <= 0 {
		poolSize = 5 // Default to 5 concurrent connections
	}

	service := &emailService{
		smtpHost:  cfg.GmailSMTP.Host,
		smtpPort:  cfg.GmailSMTP.Port,
		username:  cfg.GmailSMTP.Username,
		password:  cfg.GmailSMTP.Password,
		fromName:  cfg.GmailSMTP.FromName,
		fromEmail: cfg.GmailSMTP.FromEmail,
		rateLimiter: &rateLimiter{
			tokens:         emailsPerMinute,
			maxTokens:      emailsPerMinute,
			refillRate:     refillRate,
			lastRefillTime: time.Now(),
		},
		connPool: &smtpConnectionPool{
			connections: make(chan *smtpConnection, poolSize),
			maxSize:     poolSize,
			smtpHost:    cfg.GmailSMTP.Host,
			smtpPort:    cfg.GmailSMTP.Port,
			username:    cfg.GmailSMTP.Username,
			password:    cfg.GmailSMTP.Password,
		},
	}

	// Load templates
	if err := service.loadTemplates(cfg.Notification.TemplateDir); err != nil {
		zap.L().Warn("Failed to load email templates", zap.Error(err))
		// Don't fail initialization if templates fail to load - they're optional
	}

	zap.L().Info("EmailService initialized successfully",
		zap.String("smtp_host", service.smtpHost),
		zap.Int("pool_size", poolSize),
		zap.Int("smtp_port", service.smtpPort),
		zap.String("from_email", service.fromEmail),
		zap.Int("rate_limit", emailsPerMinute),
		zap.Duration("refill_rate", refillRate))

	return service, nil
}

// SendEmail sends an email with the specified parameters
func (s *emailService) SendEmail(ctx context.Context, to, subject string, body *string, isHTML bool) error {
	// Wait for rate limit token
	if err := s.rateLimiter.waitForToken(ctx); err != nil {
		zap.L().Warn("Rate limit wait cancelled",
			zap.String("to", to),
			zap.Error(err))
		return fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	// Validate email address
	if !isValidEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	// Create MIME headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	if isHTML {
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=\"utf-8\""
	} else {
		headers["Content-Type"] = "text/plain; charset=\"utf-8\""
	}

	// Build message
	var message bytes.Buffer
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(*body)

	// Send via SMTP with TLS
	if err := s.sendViaSMTP(to, message.Bytes()); err != nil {
		zap.L().Error("Failed to send email",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	zap.L().Info("Email sent successfully",
		zap.String("to", to),
		zap.String("subject", subject))

	return nil
}

// SendTemplatedEmail sends an email using a template
func (s *emailService) SendTemplatedEmail(ctx context.Context, to, subject, templateName string, data map[string]any) error {
	if s.templates == nil {
		return errors.New("email templates not loaded")
	}

	// Render HTML template
	var htmlBuf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&htmlBuf, templateName+".html", data); err != nil {
		return fmt.Errorf("failed to render HTML template %s: %w", templateName, err)
	}

	// Send as HTML email
	return s.SendEmail(ctx, to, subject, utils.PtrOrNil(htmlBuf.String()), true)
}

// loadTemplates loads HTML and text email templates from the specified directory
func (s *emailService) loadTemplates(templateDir string) error {
	if templateDir == "" {
		return errors.New("template directory not specified")
	}

	// Load HTML templates
	htmlPattern := filepath.Join(templateDir, "*.html")
	htmlTemplates, err := htmlTemplate.ParseGlob(htmlPattern)
	if err != nil {
		return fmt.Errorf("failed to load HTML templates: %w", err)
	}
	s.templates = htmlTemplates

	// Load text templates
	txtPattern := filepath.Join(templateDir, "*.txt")
	textTemplates, err := textTemplate.ParseGlob(txtPattern)
	if err != nil {
		zap.L().Warn("Failed to load text templates", zap.Error(err))
		// Text templates are optional fallback
	} else {
		s.textTemplates = textTemplates
	}

	zap.L().Info("Email templates loaded successfully",
		zap.String("template_dir", templateDir),
		zap.String("html_templates", htmlTemplates.DefinedTemplates()))

	return nil
}

// sendViaSMTP sends email through Gmail SMTP using connection pooling
func (s *emailService) sendViaSMTP(to string, message []byte) error {
	// Get connection from pool
	conn, err := s.connPool.getConnection(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get SMTP connection: %w", err)
	}

	// Always return or close connection when done
	returnToPool := true
	defer func() {
		if returnToPool {
			s.connPool.returnConnection(conn)
		} else {
			s.connPool.closeConnection(conn)
		}
	}()

	// Set sender
	if err = conn.client.Mail(s.fromEmail); err != nil {
		returnToPool = false // Connection might be broken
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err = conn.client.Rcpt(to); err != nil {
		returnToPool = false // Connection might be broken
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message body
	writer, err := conn.client.Data()
	if err != nil {
		returnToPool = false // Connection might be broken
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	_, err = writer.Write(message)
	if err != nil {
		writer.Close()
		returnToPool = false // Connection might be broken
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err = writer.Close(); err != nil {
		returnToPool = false // Connection might be broken
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Reset connection for reuse
	if err = conn.client.Reset(); err != nil {
		returnToPool = false // Connection might be broken
		return fmt.Errorf("failed to reset SMTP connection: %w", err)
	}

	return nil
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// Basic format check
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	// Check domain has at least one dot
	domain := parts[1]
	if !strings.Contains(domain, ".") {
		return false
	}

	// Verify domain can be resolved
	mx, err := net.LookupMX(domain)
	if err != nil || len(mx) == 0 {
		// If MX lookup fails, try A record
		ips, err := net.LookupIP(domain)
		if err != nil || len(ips) == 0 {
			return false
		}
	}

	return true
}

// waitForToken blocks until a rate limit token is available
func (rl *rateLimiter) waitForToken(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefillTime)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefillTime = now
	}

	// If we have tokens available, use one
	if rl.tokens > 0 {
		rl.tokens--
		return nil
	}

	// Calculate wait time for next token
	waitTime := rl.refillRate - (now.Sub(rl.lastRefillTime))
	if waitTime <= 0 {
		waitTime = rl.refillRate
	}

	rl.mu.Unlock()

	// Wait for token with context cancellation support
	select {
	case <-ctx.Done():
		rl.mu.Lock()
		return ctx.Err()
	case <-time.After(waitTime):
		rl.mu.Lock()
		rl.tokens = 1 // Refill happened during wait
		rl.tokens--   // Consume the token
		rl.lastRefillTime = time.Now()
		return nil
	}
}

// getConnection retrieves a connection from the pool or creates a new one
func (p *smtpConnectionPool) getConnection(_ context.Context) (*smtpConnection, error) {
	// Try to get existing connection from pool
	select {
	case conn := <-p.connections:
		// Check if connection is still valid (not older than 5 minutes)
		if time.Since(conn.lastUsed) < 5*time.Minute {
			conn.lastUsed = time.Now()
			return conn, nil
		}
		// Connection too old, close it
		if conn.client != nil {
			conn.client.Quit()
		}
		if conn.conn != nil {
			conn.conn.Close()
		}
		// Fall through to create new connection
	default:
		// No connection available, create new one
	}

	// Create new SMTP connection
	return p.createConnection()
}

// createConnection establishes a new SMTP connection
func (p *smtpConnectionPool) createConnection() (*smtpConnection, error) {
	serverAddr := net.JoinHostPort(p.smtpHost, strconv.Itoa(p.smtpPort))

	//1. Establish a plain text connection.
	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// 2. Create a new SMTP client from the plain connection.
	client, err := smtp.NewClient(conn, p.smtpHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	// 3. Check if the server supports the STARTTLS extension.
	if ok, _ := client.Extension("STARTTLS"); !ok {
		client.Quit()
		conn.Close()
		return nil, errors.New("SMTP server does not support STARTTLS")
	}

	// 4. Upgrade the connection to TLS.
	tlsConfig := &tls.Config{
		ServerName: p.smtpHost,
		MinVersion: tls.VersionTLS12,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		client.Quit()
		conn.Close()
		return nil, fmt.Errorf("failed to start TLS handshake: %w", err)
	}

	// Authenticate
	auth := smtp.PlainAuth("", p.username, p.password, p.smtpHost)
	if err = client.Auth(auth); err != nil {
		client.Quit()
		conn.Close()
		return nil, fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return &smtpConnection{
		client:    client,
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}, nil
}

// returnConnection returns a connection to the pool
func (p *smtpConnectionPool) returnConnection(conn *smtpConnection) {
	if conn == nil {
		return
	}

	// Try to return to pool (non-blocking)
	select {
	case p.connections <- conn:
		// Successfully returned to pool
	default:
		// Pool is full, close the connection
		if conn.client != nil {
			conn.client.Quit()
		}
		if conn.conn != nil {
			conn.conn.Close()
		}
	}
}

// closeConnection closes a connection without returning it to the pool
func (p *smtpConnectionPool) closeConnection(conn *smtpConnection) {
	if conn == nil {
		return
	}
	if conn.client != nil {
		conn.client.Quit()
	}
	if conn.conn != nil {
		conn.conn.Close()
	}
}

// Health returns the health status of the email service
func (s *emailService) Health(ctx context.Context) iservice_third_party.ServiceHealth {
	health := iservice_third_party.ServiceHealth{
		Name:          "EmailService",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	if s.connPool == nil {
		health.IsHealthy = false
		health.LastError = errors.New("connection pool not initialized")
		return health
	}

	// Try to get a connection from the pool (light check)
	conn, err := s.connPool.getConnection(ctx)
	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		zap.L().Debug("Email service health check failed", zap.Error(err))
	} else {
		// Return connection to pool
		s.connPool.returnConnection(conn)
		health.IsHealthy = true
		health.Details["pool_size"] = s.connPool.maxSize
		zap.L().Debug("Email service health check passed")
	}

	return health
}
