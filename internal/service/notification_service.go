package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/resend/resend-go/v3"
	"github.com/rs/zerolog/log"

	"simplehub-go/internal/crypto"
	"simplehub-go/internal/repository"
)

type NotificationService struct {
	emailRepo     *repository.EmailConfigRepository
	encryptionKey string
}

func NewNotificationService(emailRepo *repository.EmailConfigRepository, encryptionKey string) *NotificationService {
	return &NotificationService{
		emailRepo:     emailRepo,
		encryptionKey: encryptionKey,
	}
}

func (s *NotificationService) SendModelChangeNotification(siteName string, diff interface{}, fastify interface{}) {
	cfg, err := s.emailRepo.Get()
	if err != nil || !cfg.Enabled {
		return
	}

	apiKey, err := crypto.Decrypt(cfg.ResendAPIKeyEnc, s.encryptionKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to decrypt Resend API key")
		return
	}

	subject := fmt.Sprintf("【模型变更】%s - API 聚合监控", siteName)
	html := buildModelChangeHTML(siteName, diff)

	emails := parseEmails(cfg.NotifyEmails)
	if len(emails) == 0 {
		return
	}

	if err := s.sendViaResend(apiKey, emails, subject, html); err != nil {
		log.Error().Err(err).Msg("failed to send notification email")
	}
}

type SendTestEmailOptions struct {
	NotifyEmails string `json:"notifyEmails"`
	FromEmail    string `json:"fromEmail"`
}

// TEST: 发送测试邮件
func (s *NotificationService) SendTestEmail(opts *SendTestEmailOptions) error {
	cfg, err := s.emailRepo.Get()
	if err != nil {
		return fmt.Errorf("获取邮件配置失败: %w", err)
	}
	apiKey, err := crypto.Decrypt(cfg.ResendAPIKeyEnc, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("解密API密钥失败: %w", err)
	}

	emails := parseEmails(cfg.NotifyEmails)
	if opts != nil && opts.NotifyEmails != "" {
		emails = parseEmails(opts.NotifyEmails)
	}
	if len(emails) == 0 {
		return fmt.Errorf("未配置通知邮箱")
	}

	fromEmail := cfg.FromEmail
	if opts != nil && opts.FromEmail != "" {
		fromEmail = opts.FromEmail
	}

	html := `<!DOCTYPE html>
<html><head><meta charset="utf-8"></head>
<body style="font-family: sans-serif; padding: 20px;">
<h2>测试邮件</h2>
<p>这是一封来自 SimpleHub 的测试邮件，表示邮件配置正常工作。</p>
</body></html>`
	return s.sendViaResendWithFrom(apiKey, fromEmail, emails, "【测试邮件】SimpleHub 通知服务", html)
}

func (s *NotificationService) sendViaResend(apiKey string, to []string, subject, html string) error {
	from := "SimpleHub <onboarding@resend.dev>"
	if cfg, err := s.emailRepo.Get(); err == nil && cfg.FromEmail != "" {
		from = "SimpleHub <" + cfg.FromEmail + ">"
	}
	return s.sendResend(apiKey, from, to, subject, html)
}

func (s *NotificationService) sendViaResendWithFrom(apiKey, fromEmail string, to []string, subject, html string) error {
	from := "SimpleHub <" + fromEmail + ">"
	return s.sendResend(apiKey, from, to, subject, html)
}

func (s *NotificationService) sendResend(apiKey, from string, to []string, subject, html string) error {
	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      to,
		Subject: subject,
		Html:    html,
	}

	_, err := client.Emails.SendWithContext(context.Background(), params)
	if err != nil {
		return fmt.Errorf("resend API error: %w", err)
	}

	return nil
}

type SiteCheckReport struct {
	Name          string
	Error         string
	ModelCount    int
	ModelsAdded   int
	ModelsRemoved int
	CheckInOK     bool
	CheckInMsg    string
	CheckInQuota  *float64
	BillingLimit  *float64
	BillingUsage  *float64
}

func (s *NotificationService) SendAggregatedNotification(reports []SiteCheckReport) {
	cfg, err := s.emailRepo.Get()
	if err != nil || !cfg.Enabled {
		return
	}

	apiKey, err := crypto.Decrypt(cfg.ResendAPIKeyEnc, s.encryptionKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to decrypt Resend API key")
		return
	}

	subject := "【聚合检测报告】API 聚合监控"
	html := buildAggregatedHTML(reports)

	emails := parseEmails(cfg.NotifyEmails)
	if len(emails) == 0 {
		return
	}

	if err := s.sendViaResend(apiKey, emails, subject, html); err != nil {
		log.Error().Err(err).Msg("failed to send aggregated notification email")
	}
}

func buildAggregatedHTML(reports []SiteCheckReport) string {
	var successCount, failCount, changeCount int
	for _, r := range reports {
		if r.Error != "" {
			failCount++
		} else {
			successCount++
		}
		if r.ModelsAdded > 0 || r.ModelsRemoved > 0 {
			changeCount++
		}
	}

	html := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 20px;">
<div style="max-width: 600px; margin: 0 auto; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); border-radius: 8px; padding: 30px; color: white;">
<h2 style="margin: 0 0 10px;">聚合检测报告</h2>
<p style="margin: 0; opacity: 0.9;">定时检测完成</p>
</div>
<div style="max-width: 600px; margin: 20px auto 0; padding: 20px; background: #f8f9fa; border-radius: 8px;">
<p style="margin: 0 0 16px;">成功: ` + fmt.Sprintf(`<strong style="color: #22c55e;">%d</strong>`, successCount) + ` | 失败: ` + fmt.Sprintf(`<strong style="color: #ef4444;">%d</strong>`, failCount) + ` | 变更: ` + fmt.Sprintf(`<strong style="color: #f59e0b;">%d</strong>`, changeCount) + `</p>`

	for _, r := range reports {
		html += `<div style="margin: 12px 0; padding: 12px; border-radius: 6px; background: white; border: 1px solid #e5e7eb;">`
		if r.Error != "" {
			html += fmt.Sprintf(`<h4 style="margin: 0 0 4px; color: #ef4444;">❌ %s</h4><p style="margin: 0; color: #666; font-size: 13px;">%s</p>`, r.Name, r.Error)
		} else {
			icon := "✅"
			if r.ModelsAdded > 0 || r.ModelsRemoved > 0 {
				icon = "🔄"
			}
			html += fmt.Sprintf(`<h4 style="margin: 0 0 4px;">%s %s</h4>`, icon, r.Name)
			html += fmt.Sprintf(`<p style="margin: 0; color: #666; font-size: 13px;">模型: %d 个`, r.ModelCount)
			if r.ModelsAdded > 0 {
				html += fmt.Sprintf(` | <span style="color: #22c55e;">+%d</span>`, r.ModelsAdded)
			}
			if r.ModelsRemoved > 0 {
				html += fmt.Sprintf(` | <span style="color: #ef4444;">-%d</span>`, r.ModelsRemoved)
			}
			html += `</p>`
			billingLimit := float64(0)
			if r.BillingLimit != nil {
				billingLimit = *r.BillingLimit
			}
			billingUsage := float64(0)
			if r.BillingUsage != nil {
				billingUsage = *r.BillingUsage
			}
			if billingLimit > 0 || billingUsage > 0 {
				html += fmt.Sprintf(`<p style="margin: 0; color: #666; font-size: 13px;">额度: %.2f / %.2f</p>`, billingUsage, billingLimit)
			}
			if r.CheckInOK || r.CheckInMsg != "" {
				status := "✅ 签到成功"
				if !r.CheckInOK {
					status = "❌ " + r.CheckInMsg
				} else if r.CheckInQuota != nil && *r.CheckInQuota > 0 {
					status += fmt.Sprintf(" (%.2f)", *r.CheckInQuota)
				}
				html += fmt.Sprintf(`<p style="margin: 0; color: #666; font-size: 13px;">签到: %s</p>`, status)
			}
		}
		html += `</div>`
	}

	html += `<p style="margin-top: 16px; color: #666;">请登录系统查看详细变更记录。</p>
</div>
</body>
</html>`
	return html
}

func buildModelChangeHTML(siteName string, diff interface{}) string {
	html := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 20px;">
<div style="max-width: 600px; margin: 0 auto; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); border-radius: 8px; padding: 30px; color: white;">
<h2 style="margin: 0 0 10px;">模型变更通知</h2>
<p style="margin: 0; opacity: 0.9;">站点：SITE_PLACEHOLDER</p>
</div>
<div style="max-width: 600px; margin: 20px auto 0; padding: 20px; background: #f8f9fa; border-radius: 8px;">`

	if d, ok := diff.(DiffResult); ok {
		if len(d.Added) > 0 {
			html += `<h3 style="color: #22c55e;">新增模型</h3><ul style="list-style: none; padding: 0;">`
			for _, m := range d.Added {
				name := m.Name
				if name == "" {
					name = m.ID
				}
				html += fmt.Sprintf(`<li style="padding: 4px 8px; margin: 4px 0; background: #f0fdf4; border-left: 3px solid #22c55e; border-radius: 2px;">➕ %s</li>`, name)
			}
			html += `</ul>`
		}
		if len(d.Removed) > 0 {
			html += `<h3 style="color: #ef4444;">移除模型</h3><ul style="list-style: none; padding: 0;">`
			for _, m := range d.Removed {
				name := m.Name
				if name == "" {
					name = m.ID
				}
				html += fmt.Sprintf(`<li style="padding: 4px 8px; margin: 4px 0; background: #fef2f2; border-left: 3px solid #ef4444; border-radius: 2px;">➖ %s</li>`, name)
			}
			html += `</ul>`
		}
	}

html += `<p style="margin-top: 16px; color: #666;">请登录系统查看完整变更记录。</p>
</div>
</body>
</html>`
	return strings.Replace(html, "SITE_PLACEHOLDER", siteName, -1)
}

func parseEmails(raw string) []string {
	var emails []string
	if raw == "" {
		return emails
	}

	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		for _, e := range arr {
			if e != "" {
				emails = append(emails, e)
			}
		}
		return emails
	}

	for _, part := range strings.Split(raw, ",") {
		e := strings.TrimSpace(part)
		if e != "" {
			emails = append(emails, e)
		}
	}
	return emails
}
