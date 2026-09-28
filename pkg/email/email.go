package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"net/url"
	"time"
)

type EmailSender interface {
	SendActivationEmail(ctx context.Context, toEmail, recipientName, activationToken string) error
	SendResetPasswordEmail(ctx context.Context, toEmail, recipientName, resetToken string) error
}

type SMTPConfig struct {
	Host        string
	Port        string
	AuthUser    string
	AuthPass    string
	FromName    string
	FromEmail   string
	LogoURL     string
	FrontendURL string
	AppName     string
	UseTLS      bool
}

type smtpEmailSender struct {
	config SMTPConfig
}

type emailPayload struct {
	Subject       string
	Greeting      string
	Message       template.HTML
	CTAButtonText string
	CTAURL        string
	WarningNote   string
}

type templateData struct {
	Payload     emailPayload
	LogoURL     string
	AppName     string
	FrontendURL string
	Year        int
}

func NewSMTPEmailSender(cfg SMTPConfig) EmailSender {
	if cfg.AppName == "" {
		cfg.AppName = "Bookstore"
	}
	return &smtpEmailSender{
		config: cfg,
	}
}

func (s *smtpEmailSender) SendActivationEmail(ctx context.Context, toEmail, recipientName, activationToken string) error {
	activationLink := fmt.Sprintf("%s/activate-account?token=%s", s.config.FrontendURL, url.QueryEscape(activationToken))
	safeAppName := template.HTMLEscapeString(s.config.AppName)

	payload := emailPayload{
		Subject:       fmt.Sprintf("Activate Your Account - %s", s.config.AppName),
		Greeting:      fmt.Sprintf("Halo, %s!", recipientName),
		Message:       template.HTML(fmt.Sprintf("Terima kasih telah mendaftar di <strong>%s</strong>. Untuk menyelesaikan pendaftaran dan mengamankan akun Anda, silakan lakukan verifikasi email.", safeAppName)),
		CTAButtonText: "Aktivasi Akun Sekarang",
		CTAURL:        activationLink,
		WarningNote:   "Jika tombol di atas tidak berfungsi, salin dan tempel link berikut ke browser Anda. Link ini hanya berlaku selama 24 jam.",
	}

	return s.send(ctx, toEmail, recipientName, payload)
}

func (s *smtpEmailSender) SendResetPasswordEmail(ctx context.Context, toEmail, recipientName, resetToken string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.config.FrontendURL, url.QueryEscape(resetToken))
	safeAppName := template.HTMLEscapeString(s.config.AppName)

	payload := emailPayload{
		Subject:       fmt.Sprintf("Reset Kata Sandi - %s", s.config.AppName),
		Greeting:      fmt.Sprintf("Halo, %s!", recipientName),
		Message:       template.HTML(fmt.Sprintf("Kami menerima permintaan untuk mereset kata sandi akun <strong>%s</strong> Anda. Klik tombol di bawah ini untuk membuat kata sandi baru.", safeAppName)),
		CTAButtonText: "Reset Kata Sandi",
		CTAURL:        resetLink,
		WarningNote:   "Jika Anda tidak meminta reset kata sandi, silakan abaikan email ini. Link ini hanya berlaku selama 1 jam.",
	}

	return s.send(ctx, toEmail, recipientName, payload)
}

func (s *smtpEmailSender) send(ctx context.Context, toEmail, recipientName string, payload emailPayload) error {
	body, err := s.renderBaseTemplate(payload)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	fromHeader := fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromEmail)
	toHeader := fmt.Sprintf("To: %s <%s>\r\n", recipientName, toEmail)
	subjectHeader := fmt.Sprintf("Subject: %s\r\n", payload.Subject)
	mimeHeader := "MIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n"

	msg := []byte(fromHeader + toHeader + subjectHeader + mimeHeader + body)
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)

	errCh := make(chan error, 1)
	go func() {
		if s.config.UseTLS {
			errCh <- s.sendWithTLS(addr, toEmail, msg)
		} else {
			auth := smtp.PlainAuth("", s.config.AuthUser, s.config.AuthPass, s.config.Host)
			errCh <- smtp.SendMail(addr, auth, s.config.FromEmail, []string{toEmail}, msg)
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("email sending canceled: %w", ctx.Err())
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
		return nil
	}
}

func (s *smtpEmailSender) sendWithTLS(addr, toEmail string, msg []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("failed to parse SMTP address: %w", err)
	}

	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(smtp.PlainAuth("", s.config.AuthUser, s.config.AuthPass, host)); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close email writer: %w", err)
	}

	return client.Quit()
}

func (s *smtpEmailSender) renderBaseTemplate(payload emailPayload) (string, error) {
	const tmpl = `
	<!DOCTYPE html>
	<html lang="id">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>{{.Payload.Subject}}</title>
	</head>
	<body style="margin: 0; padding: 0; background-color: #f1f5f9; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; -webkit-font-smoothing: antialiased;">
		<table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="background-color: #f1f5f9; padding: 40px 10px;">
			<tr>
				<td align="center">
					<table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="max-width: 600px; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);">
						<tr>
							<td align="center" style="padding: 30px 40px; background-color: #0f172a;">
								<img src="{{.LogoURL}}" alt="Logo" width="180" style="max-width: 180px; height: auto; display: block; border: 0;" />
							</td>
						</tr>
						<tr>
							<td style="padding: 40px 40px 30px 40px; color: #334155;">
								<h2 style="margin-top: 0; margin-bottom: 16px; font-size: 22px; font-weight: 700; color: #0f172a;">
									{{.Payload.Greeting}}
								</h2>
								<p style="margin: 0 0 20px 0; font-size: 15px; line-height: 1.6; color: #475569;">
									{{.Payload.Message}}
								</p>
								<table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin: 32px auto;">
									<tr>
										<td align="center" style="border-radius: 8px; background-color: #334155;">
											<a href="{{.Payload.CTAURL}}" target="_blank" style="display: inline-block; padding: 14px 32px; font-size: 15px; font-weight: 600; color: #ffffff; text-decoration: none; border-radius: 8px; box-shadow: 0 2px 4px rgba(37,99,235,0.2);">
												{{.Payload.CTAButtonText}}
											</a>
										</td>
									</tr>
								</table>
								<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 30px 0;" />
								<p style="margin: 0 0 8px 0; font-size: 12px; color: #64748b;">
									{{.Payload.WarningNote}}
								</p>
								<div style="background-color: #f8fafc; padding: 12px 14px; border-radius: 6px; border: 1px solid #e2e8f0; word-break: break-all;">
									<a href="{{.Payload.CTAURL}}" style="font-size: 12px; color: #334155; text-decoration: underline;">{{.Payload.CTAURL}}</a>
								</div>
							</td>
						</tr>
						<tr>
							<td style="padding: 30px 40px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; text-align: center;">
								<p style="margin: 0 0 12px 0; font-size: 13px; color: #64748b;">
									<a href="{{.FrontendURL}}/help" style="color: #64748b; text-decoration: underline; margin: 0 8px;">Pusat Bantuan</a> &bull;
									<a href="{{.FrontendURL}}/privacy" style="color: #64748b; text-decoration: underline; margin: 0 8px;">Kebijakan Privasi</a> &bull;
									<a href="{{.FrontendURL}}/terms" style="color: #64748b; text-decoration: underline; margin: 0 8px;">Syarat & Ketentuan</a>
								</p>
								<p style="margin: 0 0 12px 0; font-size: 12px; line-height: 1.5; color: #94a3b8;">
									{{.AppName}} Inc. &bull; Email ini dikirim secara otomatis.
								</p>
								<p style="margin: 0; font-size: 12px; color: #94a3b8;">
									&copy; {{.Year}} {{.AppName}}. All rights reserved.
								</p>
							</td>
						</tr>
					</table>
				</td>
			</tr>
		</table>
	</body>
	</html>`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := templateData{
		Payload:     payload,
		LogoURL:     s.config.LogoURL,
		AppName:     s.config.AppName,
		FrontendURL: s.config.FrontendURL,
		Year:        time.Now().Year(),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
