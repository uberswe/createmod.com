package mailer

import (
	"fmt"
	"html"
	"strings"
)

// ModerationEmail describes a schematic moderation notification. Rendered by
// ModerationEmailHTML into an email-safe, light-theme layout: logo top-left, a
// 3px accent bar in the state colour, heading + paragraph, an optional white
// checklist box, a solid gold CTA, and a small footer. (#1646)
type ModerationEmail struct {
	Accent    string   // state colour for the 3px bar + heading (hex)
	Heading   string   // e.g. "Your schematic Steam Train is live"
	Body      string   // one paragraph (plain text; newlines -> <br>)
	Checklist []string // optional white checklist box items
	CTALabel  string   // optional gold button label
	CTAURL    string   // optional gold button URL
	Footer    string   // optional small note under the button
}

// Moderation state accent colours (email-safe hex).
const (
	moderationGreen  = "#2f9e44"
	moderationYellow = "#c98a00"
	moderationBlue   = "#3b82c4"
	moderationRed    = "#d63939"
	moderationGold   = "#9e7735"
)

// ModerationEmailHTML renders a ModerationEmail to a full HTML document.
func ModerationEmailHTML(m ModerationEmail) string {
	accent := m.Accent
	if accent == "" {
		accent = moderationGold
	}
	body := strings.ReplaceAll(html.EscapeString(m.Body), "\n", "<br>")

	checklistBlock := ""
	if len(m.Checklist) > 0 {
		var rows strings.Builder
		for _, item := range m.Checklist {
			rows.WriteString(fmt.Sprintf(
				`<tr><td style="padding:6px 0;font-size:14px;color:#334155;line-height:1.5">&bull;&nbsp;%s</td></tr>`,
				strings.ReplaceAll(html.EscapeString(item), "\n", "<br>")))
		}
		checklistBlock = fmt.Sprintf(`<tr><td style="padding:16px 0 0 0">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f8fafc;border:1px solid #e2e8f0;border-radius:8px">
<tr><td style="padding:14px 16px">
<table width="100%%" cellpadding="0" cellspacing="0">%s</table>
</td></tr></table></td></tr>`, rows.String())
	}

	ctaBlock := ""
	if m.CTALabel != "" && m.CTAURL != "" {
		ctaBlock = fmt.Sprintf(`<tr><td style="padding:22px 0 0 0">
<a href="%s" style="display:inline-block;padding:11px 26px;background-color:%s;color:#ffffff;text-decoration:none;border-radius:6px;font-weight:bold;font-size:14px">%s</a>
</td></tr>`, html.EscapeString(m.CTAURL), moderationGold, html.EscapeString(m.CTALabel))
	}

	footerNote := ""
	if m.Footer != "" {
		footerNote = fmt.Sprintf(`<tr><td style="padding:18px 0 0 0;font-size:12.5px;color:#94a3b8;line-height:1.5">%s</td></tr>`,
			strings.ReplaceAll(html.EscapeString(m.Footer), "\n", "<br>"))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background-color:#f4f6fa;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f6fa;padding:32px 0">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:10px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.08)">
<tr><td style="height:3px;background-color:%s;line-height:3px;font-size:0">&nbsp;</td></tr>
<tr><td style="padding:20px 28px 4px 28px">
<span style="color:#9e7735;font-size:18px;font-weight:bold">CreateMod.com</span>
</td></tr>
<tr><td style="padding:12px 28px 28px 28px">
<table width="100%%" cellpadding="0" cellspacing="0">
<tr><td style="padding:0 0 12px 0">
<h1 style="margin:0;font-size:19px;color:%s">%s</h1>
</td></tr>
<tr><td style="padding:0;font-size:14px;line-height:1.6;color:#475569">%s</td></tr>
%s
%s
%s
</table>
</td></tr>
<tr><td style="padding:16px 28px;text-align:center;font-size:12px;color:#94a3b8;border-top:1px solid #eef2f6">
CreateMod.com &mdash; Minecraft Create mod schematics
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`, accent, accent, html.EscapeString(m.Heading), body, checklistBlock, ctaBlock, footerNote)
}

// SchematicLiveEmail: a schematic reached full visibility. (green)
func SchematicLiveEmail(title, schematicURL string) string {
	return ModerationEmailHTML(ModerationEmail{
		Accent:   moderationGreen,
		Heading:  fmt.Sprintf("Your schematic %s is live", title),
		Body:     "Everything checks out — your schematic is now fully published and appears in Latest and search. Thanks for sharing your build!",
		CTALabel: "View your schematic",
		CTAURL:   schematicURL,
	})
}

// SchematicActionNeededEmail: published with limits; the checklist unlocks full
// visibility. (yellow)
func SchematicActionNeededEmail(title, schematicURL string, checklist []string) string {
	return ModerationEmailHTML(ModerationEmail{
		Accent:    moderationYellow,
		Heading:   fmt.Sprintf("Action needed: unlock full visibility for %s", title),
		Body:      "Your schematic is published and reachable at its link, but it stays out of Latest and search until you resolve the note below. Fixing it promotes it automatically — no re-submission needed.",
		Checklist: checklist,
		CTALabel:  "Fix it now",
		CTAURL:    schematicURL,
		Footer:    "No deadline. Your schematic stays live at its link meanwhile.",
	})
}

// SchematicImageReviewEmail: one or more images are being reviewed. (blue)
func SchematicImageReviewEmail(title, schematicURL string, slaHours int) string {
	return ModerationEmailHTML(ModerationEmail{
		Accent:   moderationBlue,
		Heading:  fmt.Sprintf("One image on %s is being reviewed", title),
		Body:     fmt.Sprintf("Your schematic is published. One image was flagged by the automated scanner and is hidden while a human takes a look — this usually happens within %d hours. Shaders sometimes confuse the scanner; this is normal. Visitors don't see the hidden image meanwhile, and it returns automatically if approved.", slaHours),
		CTALabel: "View your schematic",
		CTAURL:   schematicURL,
		Footer:   "If the image is removed, we'll tell you exactly why and you can upload a replacement.",
	})
}

// SchematicNotPublishedEmail: a schematic was rejected. fixable toggles the
// second-chance vs appeal-only copy. (red)
func SchematicNotPublishedEmail(title, schematicURL, reason string, fixable bool) string {
	body := fmt.Sprintf("%s was not published because it broke a content rule.", title)
	if reason != "" {
		body += " " + reason
	}
	m := ModerationEmail{
		Accent:   moderationRed,
		Heading:  fmt.Sprintf("%s was not published", title),
		CTALabel: "Open appeal thread",
		CTAURL:   schematicURL,
	}
	if fixable {
		m.Body = body + " You can fix the issue and resubmit once — see the checklist on your schematic page."
		m.CTALabel = "Fix and resubmit"
		m.Footer = "Questions? Reply in the moderation thread on your schematic page."
	} else {
		m.Body = body + " A human confirmed this decision, so it won't be republished."
		m.Footer = "Fixable rejections instead include a checklist and a one-time resubmit button."
	}
	return ModerationEmailHTML(m)
}
