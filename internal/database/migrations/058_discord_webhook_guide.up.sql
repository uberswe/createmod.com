-- Seed a guide explaining how to set up Discord webhooks for schematic notifications.

INSERT INTO guides (id, author_id, title, description, content, slug, upload_link)
VALUES (
    'guide_discord_webhook_001',
    NULL,
    'How to Set Up Discord Webhook Notifications',
    'Learn how to receive real-time notifications in your Discord server whenever new schematics are uploaded to CreateMod.com.',
    '<h2>What Are Discord Webhooks?</h2>
<p>Discord webhooks let you receive automatic messages in a Discord channel when something happens on another service. On CreateMod.com, you can set up a webhook so that your Discord server gets a notification whenever a new schematic is uploaded, helping your community stay up to date with the latest builds.</p>

<h2>Step 1: Create a Webhook in Discord</h2>
<ol>
<li>Open your Discord server and navigate to the channel where you want notifications to appear.</li>
<li>Click the <strong>gear icon</strong> next to the channel name to open <strong>Channel Settings</strong>.</li>
<li>Select <strong>Integrations</strong> from the left sidebar.</li>
<li>Click <strong>Webhooks</strong>, then <strong>New Webhook</strong>.</li>
<li>Give your webhook a name (e.g., "CreateMod Notifications") and optionally set an avatar.</li>
<li>Click <strong>Copy Webhook URL</strong> — you''ll need this in the next step.</li>
</ol>
<p><strong>Important:</strong> Keep your webhook URL private. Anyone with this URL can send messages to your channel.</p>

<h2>Step 2: Add the Webhook to CreateMod.com</h2>
<ol>
<li>Log in to your CreateMod.com account.</li>
<li>Go to <strong>Settings</strong> by clicking your avatar in the top-right corner.</li>
<li>Select the <strong>Webhooks</strong> tab from the settings sidebar.</li>
<li>Click <strong>Add Webhook</strong>.</li>
<li>Paste the Discord webhook URL you copied in Step 1.</li>
<li>Click <strong>Save</strong>. The system will validate that the URL is a valid Discord webhook.</li>
</ol>

<h2>Step 3: Choose What Triggers Notifications</h2>
<p>Once your webhook is saved, notifications will be sent for new schematic uploads. Each notification includes:</p>
<ul>
<li>The <strong>schematic title</strong> and a link to view it.</li>
<li>The <strong>author</strong> who uploaded it.</li>
<li>A <strong>thumbnail</strong> preview of the build.</li>
<li>The <strong>category</strong> and any tags.</li>
</ul>

<h2>Troubleshooting</h2>
<h3>Webhook URL rejected</h3>
<p>Make sure the URL starts with <code>https://discord.com/api/webhooks/</code> or <code>https://discordapp.com/api/webhooks/</code>. Other URLs are not accepted for security reasons.</p>

<h3>Not receiving notifications</h3>
<ul>
<li>Check that the webhook still exists in your Discord channel settings — if it was deleted, you''ll need to create a new one.</li>
<li>Verify the webhook is listed as <strong>Active</strong> on your CreateMod.com Webhooks settings page.</li>
<li>Make sure the bot/webhook has permission to post in the target channel.</li>
</ul>

<h3>Too many notifications</h3>
<p>If your server is receiving too many messages, consider creating a dedicated <code>#schematic-feed</code> channel for webhook notifications so they don''t clutter your main chat.</p>

<h2>Managing Your Webhooks</h2>
<p>You can manage your webhooks at any time from the <a href="/settings/webhooks">Webhooks settings page</a>:</p>
<ul>
<li><strong>Test</strong> — Send a test message to verify the webhook is working.</li>
<li><strong>Delete</strong> — Remove a webhook to stop receiving notifications.</li>
</ul>',
    'discord-webhook-notifications',
    ''
) ON CONFLICT (id) DO NOTHING;
