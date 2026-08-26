package mailer

import (
	"fmt"
	"html"
	"strings"
)

// ModerationEmail describes a schematic moderation notification. Rendered by
// ModerationEmailHTML into an email-safe DARK layout matching the site: logo
// top-left, a 3px accent bar in the state colour, heading + paragraph, an
// optional checklist box, a solid gold CTA, and a small footer. (#1646)
type ModerationEmail struct {
	Accent    string   // state colour for the 3px bar + heading (hex)
	Heading   string   // e.g. "Your schematic Steam Train is live"
	Body      string   // one paragraph (plain text; newlines -> <br>)
	Checklist []string // optional checklist box items
	// Note is an optional callout under the checklist, e.g. telling the author an
	// automated review may be wrong and they can ask for a human. (#1646)
	Note     string
	CTALabel string // optional gold button label
	CTAURL   string // optional gold button URL
	Footer   string // optional small note under the button
}

// automatedReviewNote is shown on emails whose feedback came from the automated
// review, so authors know they can escalate to a human. (#1646)
const automatedReviewNote = "Our automated review sometimes makes mistakes, if you believe the message above is wrong you can submit your schematic for human review without changes."

// Moderation email colours. Dark scheme matching the site (page #1f2121, card
// #2d3030, gold #bf9045, muted text), plus email-safe accent hexes.
const (
	moderationGreen  = "#2fb344"
	moderationYellow = "#f59f00"
	moderationBlue   = "#4299e1"
	moderationRed    = "#e0645a"
	moderationGold   = "#bf9045"

	emailBg       = "#17191a" // outer background
	emailCard     = "#242727" // card background
	emailBox      = "#1d2020" // checklist box background
	emailBorder   = "rgba(255,255,255,0.10)"
	emailText     = "#dce1e7" // body text
	emailMuted    = "#9ea5ad" // footer/muted
	emailHeading  = "#f1ead9"
	emailGoldText = "#20160a" // text on gold button
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
				`<tr><td style="padding:6px 0;font-size:14px;color:%s;line-height:1.5"><span style="color:%s">&bull;</span>&nbsp;%s</td></tr>`,
				emailText, moderationGold, strings.ReplaceAll(html.EscapeString(item), "\n", "<br>")))
		}
		checklistBlock = fmt.Sprintf(`<tr><td style="padding:16px 0 0 0">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:%s;border:1px solid %s;border-radius:8px">
<tr><td style="padding:14px 16px">
<table width="100%%" cellpadding="0" cellspacing="0">%s</table>
</td></tr></table></td></tr>`, emailBox, emailBorder, rows.String())
	}

	noteBlock := ""
	if m.Note != "" {
		noteBlock = fmt.Sprintf(`<tr><td style="padding:16px 0 0 0">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:%s;border:1px solid %s;border-radius:8px">
<tr><td style="padding:12px 16px;font-size:13px;line-height:1.5;color:%s">%s</td></tr>
</table></td></tr>`, emailBox, emailBorder, emailMuted, strings.ReplaceAll(html.EscapeString(m.Note), "\n", "<br>"))
	}

	ctaBlock := ""
	if m.CTALabel != "" && m.CTAURL != "" {
		ctaBlock = fmt.Sprintf(`<tr><td style="padding:22px 0 0 0">
<a href="%s" style="display:inline-block;padding:11px 26px;background-color:%s;color:%s;text-decoration:none;border-radius:6px;font-weight:bold;font-size:14px">%s</a>
</td></tr>`, html.EscapeString(m.CTAURL), moderationGold, emailGoldText, html.EscapeString(m.CTALabel))
	}

	footerNote := ""
	if m.Footer != "" {
		footerNote = fmt.Sprintf(`<tr><td style="padding:18px 0 0 0;font-size:12.5px;color:%s;line-height:1.5">%s</td></tr>`,
			emailMuted, strings.ReplaceAll(html.EscapeString(m.Footer), "\n", "<br>"))
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
<tr><td style="padding:0;font-size:14px;line-height:1.6;color:%s">%s</td></tr>
%s
%s
%s
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
</html>`, emailBg, emailBg, emailCard, emailBorder, accent, moderationGold, emailHeading, html.EscapeString(m.Heading), emailText, body, checklistBlock, noteBlock, ctaBlock, footerNote, emailMuted, emailBorder)
}

// SchematicLiveEmail: a schematic reached full visibility. (green)
func SchematicLiveEmail(title, schematicURL string) string {
	return ModerationEmailHTML(ModerationEmail{
		Accent:   moderationGreen,
		Heading:  fmt.Sprintf("Your schematic %s is live", title),
		Body:     "Everything passed and your schematic is fully published. It shows up in Latest and in search like any other build. Thanks for sharing it!",
		CTALabel: "View your schematic",
		CTAURL:   schematicURL,
	})
}

// SchematicActionNeededEmail: published with limits; the checklist unlocks full
// visibility. (yellow) When automated, includes the note telling the author they
// can escalate to a human review. (#1646)
func SchematicActionNeededEmail(title, schematicURL string, checklist []string, automated bool) string {
	m := ModerationEmail{
		Accent:    moderationYellow,
		Heading:   fmt.Sprintf("Action needed: unlock full visibility for %s", title),
		Body:      "Your schematic is published and users can view it via the link, but it is not shown anywhere on the website until you fix the following issues.",
		Checklist: checklist,
		CTALabel:  "Fix it now",
		CTAURL:    schematicURL,
		Footer:    "Your schematic is visible via direct link only.",
	}
	if automated {
		m.Note = automatedReviewNote
	}
	return ModerationEmailHTML(m)
}

// SchematicImageReviewEmail: one or more images are being reviewed. (blue)
func SchematicImageReviewEmail(title, schematicURL string, slaHours int) string {
	return ModerationEmailHTML(ModerationEmail{
		Accent:   moderationBlue,
		Heading:  fmt.Sprintf("One image on %s is being reviewed", title),
		Body:     fmt.Sprintf("Your schematic is live. One of your images tripped the automated scanner, so we've hidden it while a person takes a look, usually within %d hours. Shaders throw the scanner off sometimes, so this is often a false alarm. Visitors don't see the hidden image, and it comes back on its own if it passes.", slaHours),
		CTALabel: "View your schematic",
		CTAURL:   schematicURL,
		Footer:   "If we do remove it, we'll tell you exactly why so you can upload a replacement.",
	})
}

// SchematicNotPublishedEmail: a schematic was rejected. fixable toggles the
// second-chance vs appeal-only copy. (red)
func SchematicNotPublishedEmail(title, schematicURL, reason string, fixable bool) string {
	body := fmt.Sprintf("%s wasn't published because it broke one of our content rules.", title)
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
		m.Body = body + " You can fix it and resubmit once. The checklist on your schematic page tells you what to change."
		m.CTALabel = "Fix and resubmit"
		m.Footer = "Got questions? Reply in the moderation thread on your schematic page."
	} else {
		m.Body = body + " A moderator reviewed this and it won't go back up."
		m.Footer = "If a rejection is fixable, we include a checklist and a one-time resubmit button instead."
	}
	return ModerationEmailHTML(m)
}
