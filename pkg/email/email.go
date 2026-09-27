package email

import (
	"context"
	"fmt"
	"net/smtp"
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
}

type smtpEmailSender struct {
	config SMTPConfig
}

type emailPayload struct {
	Subject       string
	Greeting      string
	Message       string
	CTAButtonText string
	CTAURL        string
	WarningNote   string
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
	activationLink := fmt.Sprintf("%s/activate-account?token=%s", s.config.FrontendURL, activationToken)

	payload := emailPayload{
		Subject:       fmt.Sprintf("Activate Your Account - %s", s.config.AppName),
		Greeting:      fmt.Sprintf("Halo, %s!", recipientName),
		Message:       fmt.Sprintf("Terima kasih telah mendaftar di <strong>%s</strong>. Untuk menyelesaikan pendaftaran dan mengamankan akun Anda, silakan lakukan verifikasi email.", s.config.AppName),
		CTAButtonText: "Aktivasi Akun Sekarang",
		CTAURL:        activationLink,
		WarningNote:   "Jika tombol di atas tidak berfungsi, salin dan tempel link berikut ke browser Anda. Link ini hanya berlaku selama 24 jam.",
	}

	return s.send(ctx, toEmail, recipientName, payload)
}

func (s *smtpEmailSender) SendResetPasswordEmail(ctx context.Context, toEmail, recipientName, resetToken string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.config.FrontendURL, resetToken)

	payload := emailPayload{
		Subject:       fmt.Sprintf("Reset Kata Sandi - %s", s.config.AppName),
		Greeting:      fmt.Sprintf("Halo, %s!", recipientName),
		Message:       fmt.Sprintf("Kami menerima permintaan untuk mereset kata sandi akun <strong>%s</strong> Anda. Klik tombol di bawah ini untuk membuat kata sandi baru.", s.config.AppName),
		CTAButtonText: "Reset Kata Sandi",
		CTAURL:        resetLink,
		WarningNote:   "Jika Anda tidak meminta reset kata sandi, silakan abaikan email ini. Link ini hanya berlaku selama 1 jam.",
	}

	return s.send(ctx, toEmail, recipientName, payload)
}

func (s *smtpEmailSender) send(ctx context.Context, toEmail, recipientName string, payload emailPayload) error {
	body := s.renderBaseTemplate(payload)
	fromHeader := fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromEmail)
	toHeader := fmt.Sprintf("To: %s <%s>\r\n", recipientName, toEmail)
	subjectHeader := fmt.Sprintf("Subject: %s\r\n", payload.Subject)
	mimeHeader := "MIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n"

	msg := []byte(fromHeader + toHeader + subjectHeader + mimeHeader + body)
	auth := smtp.PlainAuth("", s.config.AuthUser, s.config.AuthPass, s.config.Host)
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, s.config.FromEmail, []string{toEmail}, msg)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("Email sending canceled: %w", ctx.Err())
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("Failed to send email: %w", err)
		}
		return nil
	}
}

func (s *smtpEmailSender) renderBaseTemplate(payload emailPayload) string {
	return fmt.Sprintf(`
				<!DOCTYPE html>
				<html lang="id">
				<head>
						<meta charset="UTF-8">
						<meta name="viewport" content="width=device-width, initial-scale=1.0">
						<title>%s</title>
				</head>
				<body style="margin: 0; padding: 0; background-color: #f1f5f9; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; -webkit-font-smoothing: antialiased;">
						<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background-color: #f1f5f9; padding: 40px 10px;">
								<tr>
										<td align="center">
												<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width: 600px; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);">
														<tr>
																<td align="center" style="padding: 30px 40px; background-color: #0f172a;">
																		<img src="%s" alt="Logo" width="180" style="max-width: 180px; height: auto; display: block; border: 0;" />
																</td>
														</tr>

														<tr>
																<td style="padding: 40px 40px 30px 40px; color: #334155;">
																		<h2 style="margin-top: 0; margin-bottom: 16px; font-size: 22px; font-weight: 700; color: #0f172a;">
																				%s
																		</h2>
																		<p style="margin: 0 0 20px 0; font-size: 15px; line-height: 1.6; color: #475569;">
																				%s
																		</p>

																		<table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin: 32px auto;">
																				<tr>
																						<td align="center" style="border-radius: 8px; background-color: #334155;">
																								<a href="%s" target="_blank" style="display: inline-block; padding: 14px 32px; font-size: 15px; font-weight: 600; color: #ffffff; text-decoration: none; border-radius: 8px; box-shadow: 0 2px 4px rgba(37,99,235,0.2);">
																										%s
																								</a>
																						</td>
																				</tr>
																		</table>

																		<hr style="border: none; border-top: 1px solid #e2e8f0; margin: 30px 0;" />

																		<p style="margin: 0 0 8px 0; font-size: 12px; color: #64748b;">
																				%s
																		</p>
																		<div style="background-color: #f8fafc; padding: 12px 14px; border-radius: 6px; border: 1px solid #e2e8f0; word-break: break-all;">
																				<a href="%s" style="font-size: 12px; color: #334155; text-decoration: underline;">%s</a>
																		</div>
																</td>
														</tr>

														<tr>
																<td style="padding: 30px 40px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; text-align: center;">
																		<p style="margin: 0 0 12px 0; font-size: 13px; color: #64748b;">
																				<a href="%s/help" style="color: #64748b; text-decoration: underline; margin: 0 8px;">Pusat Bantuan</a> &bull;
																				<a href="%s/privacy" style="color: #64748b; text-decoration: underline; margin: 0 8px;">Kebijakan Privasi</a> &bull;
																				<a href="%s/terms" style="color: #64748b; text-decoration: underline; margin: 0 8px;">Syarat & Ketentuan</a>
																		</p>

																		<p style="margin: 0 0 12px 0; font-size: 12px; line-height: 1.5; color: #94a3b8;">
																				%s Inc. &bull; Jl. Jendral Sudirman No. 123, Jakarta Selatan, Indonesia<br />
																				Email ini dikirim secara otomatis. Mohon tidak membalas email ini.
																		</p>

																		<p style="margin: 0; font-size: 12px; color: #94a3b8;">
																				&copy; %d %s. All rights reserved.
																		</p>
																</td>
														</tr>
												</table>
										</td>
								</tr>
						</table>
				</body>
				</html>
				`,
		payload.Subject,
		s.config.LogoURL,
		payload.Greeting,
		payload.Message,
		payload.CTAURL,
		payload.CTAButtonText,
		payload.WarningNote,
		payload.CTAURL,
		payload.CTAURL,
		s.config.FrontendURL,
		s.config.FrontendURL,
		s.config.FrontendURL,
		s.config.AppName,
		time.Now().Year(),
		s.config.AppName,
	)
}
