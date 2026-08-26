package mailer

import (
	"fmt"
	"html"
	"strings"
)

// genericEmailHTML renders a transactional/notification email in the same dark,
// gold-accented layout as the moderation emails (shared palette constants in
// moderation_email.go), so every email the app sends looks like one product.
// When escapeBody is true the body is HTML-escaped (newlines -> <br>); otherwise
// it is inserted as trusted HTML. An empty imageURL omits the image, an empty
// linkURL omits the button (buttonLabel defaults to "View"). (#1646)
func genericEmailHTML(title, imageURL, linkURL, buttonLabel, body string, escapeBody bool) string {
	escapedTitle := html.EscapeString(title)
	if escapeBody {
		body = strings.ReplaceAll(html.EscapeString(body), "\n", "<br>")
	}

	imageBlock := ""
	if imageURL != "" {
		imageBlock = fmt.Sprintf(`<tr><td style="padding:16px 0 0 0">
<img src="%s" alt="%s" style="max-width:100%%;height:auto;border-radius:8px;display:block" />
</td></tr>`, imageURL, escapedTitle)
	}

	linkBlock := ""
	if linkURL != "" {
		if buttonLabel == "" {
			buttonLabel = "View"
		}
		linkBlock = fmt.Sprintf(`<tr><td style="padding:22px 0 0 0">
<a href="%s" style="display:inline-block;padding:11px 26px;background-color:%s;color:%s;text-decoration:none;border-radius:6px;font-weight:bold;font-size:14px">%s</a>
</td></tr>`, html.EscapeString(linkURL), moderationGold, emailGoldText, html.EscapeString(buttonLabel))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="dark"><meta name="supported-color-schemes" content="dark"></head>
<body style="margin:0;padding:0;background-color:%s;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:%s;padding:32px 0">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background-color:%s;border:1px solid %s;border-radius:10px;overflow:hidden">
<tr><td style="height:3px;background-color:%s;line-height:3px;font-size:0">&nbsp;</td></tr>
<tr><td style="padding:20px 28px 4px 28px">
<span style="color:%s;font-size:18px;font-weight:bold">CreateMod.com</span>
</td></tr>
<tr><td style="padding:12px 28px 28px 28px">
<table width="100%%" cellpadding="0" cellspacing="0">
<tr><td style="padding:0 0 12px 0">
<h1 style="margin:0;font-size:19px;color:%s">%s</h1>
</td></tr>
%s
<tr><td style="padding:0;font-size:14px;line-height:1.6;color:%s">%s</td></tr>
%s
</table>
</td></tr>
<tr><td style="padding:16px 28px;text-align:center;font-size:12px;color:%s;border-top:1px solid %s">
CreateMod.com &bull; Minecraft Create mod schematics
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`, emailBg, emailBg, emailCard, emailBorder, moderationGold, moderationGold, emailHeading, escapedTitle, imageBlock, emailText, body, linkBlock, emailMuted, emailBorder)
}

// EmailHTML builds a branded HTML email body from plain text. If imageURL is
// empty, the image is omitted. If linkURL is empty, the button is omitted.
// buttonLabel defaults to "View" if empty.
func EmailHTML(title, imageURL, linkURL, buttonLabel, bodyText string) string {
	return genericEmailHTML(title, imageURL, linkURL, buttonLabel, bodyText, true)
}

// EmailHTMLRaw is like EmailHTML but bodyHTML is inserted as-is (no escaping).
// Use when the caller builds trusted HTML (e.g. verification code with <strong>).
func EmailHTMLRaw(title, imageURL, linkURL, buttonLabel, bodyHTML string) string {
	return genericEmailHTML(title, imageURL, linkURL, buttonLabel, bodyHTML, false)
}

// SchematicEmailHTML is a convenience wrapper for schematic-related emails.
func SchematicEmailHTML(title, imageURL, schematicURL, bodyText string) string {
	return EmailHTML(title, imageURL, schematicURL, "View Schematic", bodyText)
}
