package mailer

import (
	"fmt"
	"html"
)

// SchematicEmailHTML builds a formatted HTML email body for schematic-related
// notifications. If imageURL is empty, the image section is omitted.
func SchematicEmailHTML(title, imageURL, schematicURL, bodyText string) string {
	escapedTitle := html.EscapeString(title)
	escapedBody := html.EscapeString(bodyText)

	imageBlock := ""
	if imageURL != "" {
		imageBlock = fmt.Sprintf(`<tr><td style="padding:0 0 16px 0;text-align:center">
<img src="%s" alt="%s" style="max-width:100%%;height:auto;border-radius:8px" />
</td></tr>`, imageURL, escapedTitle)
	}

	linkBlock := ""
	if schematicURL != "" {
		linkBlock = fmt.Sprintf(`<tr><td style="padding:16px 0 0 0;text-align:center">
<a href="%s" style="display:inline-block;padding:10px 24px;background-color:#206bc4;color:#ffffff;text-decoration:none;border-radius:6px;font-weight:bold">View Schematic</a>
</td></tr>`, schematicURL)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background-color:#f4f6fa;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f6fa;padding:32px 0">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1)">
<tr><td style="background-color:#206bc4;padding:20px 24px;text-align:center">
<span style="color:#ffffff;font-size:20px;font-weight:bold">CreateMod.com</span>
</td></tr>
<tr><td style="padding:24px">
<table width="100%%" cellpadding="0" cellspacing="0">
<tr><td style="padding:0 0 16px 0">
<h2 style="margin:0;font-size:18px;color:#1e293b">%s</h2>
</td></tr>
%s
<tr><td style="padding:0;font-size:14px;line-height:1.6;color:#475569">
%s
</td></tr>
%s
</table>
</td></tr>
<tr><td style="padding:16px 24px;text-align:center;font-size:12px;color:#94a3b8;border-top:1px solid #e2e8f0">
CreateMod.com &mdash; Minecraft Create mod schematics
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`, escapedTitle, imageBlock, escapedBody, linkBlock)
}
