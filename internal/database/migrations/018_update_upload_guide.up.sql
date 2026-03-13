-- Update the upload guide to mention the Create: Schematic Upload mod as an easier alternative.
UPDATE guides
SET content = '<h2>Easiest Method: Create Schematic Upload Mod</h2>
<div style="display:flex;gap:12px;align-items:flex-start;margin-bottom:16px;">
<img src="/assets/x/icons/create-schematic-upload.png" alt="Create: Schematic Upload" width="48" height="48" style="border-radius:50%;flex-shrink:0;">
<div>
<p style="margin:0 0 12px 0;">The <strong>Create: Schematic Upload</strong> mod lets you upload schematics directly from Minecraft to createmod.com. Save a schematic in-game and a shareable link appears in chat instantly &mdash; no manual file handling needed.</p>
<div style="display:flex;flex-wrap:wrap;gap:8px;">
<a href="https://www.curseforge.com/minecraft/mc-mods/create-schematic-upload" target="_blank" rel="noopener" style="display:inline-flex;align-items:center;gap:6px;padding:6px 16px;border-radius:4px;background:#F16436;color:#fff;text-decoration:none;font-weight:500;font-size:14px;"><svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg"><path d="M18.326 9.2145S23.2261 8.4418 24 6.1882h-7.5066V4.4H0l2.0318 2.3576V9.173s5.1267-.2665 7.1098 1.2372c2.7146 2.516-3.053 5.917-3.053 5.917L5.0995 19.6c1.5465-1.4726 4.494-3.3775 9.8983-3.2857-2.0565.65-4.1245 1.6651-5.7344 3.2857h10.9248l-1.0288-3.2726s-7.918-4.6688-.8336-7.1127z"/></svg> CurseForge</a>
<a href="https://modrinth.com/mod/create-schematic-upload" target="_blank" rel="noopener" style="display:inline-flex;align-items:center;gap:6px;padding:6px 16px;border-radius:4px;background:#1BD96A;color:#fff;text-decoration:none;font-weight:500;font-size:14px;"><svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg"><path d="M12.252.004a11.78 11.768 0 0 0-8.92 3.73 11 10.999 0 0 0-2.17 3.11 11.37 11.359 0 0 0-1.16 5.169c0 1.42.17 2.5.6 3.77.24.759.77 1.899 1.17 2.529a12.3 12.298 0 0 0 8.85 5.639c.44.05 2.54.07 2.76.02.2-.04.22.1-.26-1.7l-.36-1.37-1.01-.06a8.5 8.489 0 0 1-5.18-1.8 5.34 5.34 0 0 1-1.3-1.26c0-.05.34-.28.74-.5a37.572 37.545 0 0 1 2.88-1.629c.03 0 .5.45 1.06.98l1 .97 2.07-.43 2.06-.43 1.47-1.47c.8-.8 1.48-1.5 1.48-1.52 0-.09-.42-1.63-.46-1.7-.04-.06-.2-.03-1.02.18-.53.13-1.2.3-1.45.4l-.48.15-.53.53-.53.53-.93.1-.93.07-.52-.5a2.7 2.7 0 0 1-.96-1.7l-.13-.6.43-.57c.68-.9.68-.9 1.46-1.1.4-.1.65-.2.83-.33.13-.099.65-.579 1.14-1.069l.9-.9-.7-.7-.7-.7-1.95.54c-1.07.3-1.96.53-1.97.53-.03 0-2.23 2.48-2.63 2.97l-.29.35.28 1.03c.16.56.3 1.16.31 1.34l.03.3-.34.23c-.37.23-2.22 1.3-2.84 1.63-.36.2-.37.2-.44.1-.08-.1-.23-.6-.32-1.03-.18-.86-.17-2.75.02-3.73a8.84 8.839 0 0 1 7.9-6.93c.43-.03.77-.08.78-.1.06-.17.5-2.999.47-3.039-.01-.02-.1-.02-.2-.03Zm3.68.67c-.2 0-.3.1-.37.38-.06.23-.46 2.42-.46 2.52 0 .04.1.11.22.16a8.51 8.499 0 0 1 2.99 2 8.38 8.379 0 0 1 2.16 3.449 6.9 6.9 0 0 1 .4 2.8c0 1.07 0 1.27-.1 1.73a9.37 9.369 0 0 1-1.76 3.769c-.32.4-.98 1.06-1.37 1.38-.38.32-1.54 1.1-1.7 1.14-.1.03-.1.06-.07.26.03.18.64 2.56.7 2.78l.06.06a12.07 12.058 0 0 0 7.27-9.4c.13-.77.13-2.58 0-3.4a11.96 11.948 0 0 0-5.73-8.578c-.7-.42-2.05-1.06-2.25-1.06Z"/></svg> Modrinth</a>
<a href="https://github.com/uberswe/CreateSchematicUpload" target="_blank" rel="noopener" style="display:inline-flex;align-items:center;gap:6px;padding:6px 16px;border-radius:4px;background:#24292e;color:#fff;text-decoration:none;font-weight:500;font-size:14px;"><svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg"><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg> GitHub</a>
</div>
</div>
</div>
<p>Once installed, the workflow is:</p>
<ol>
<li>Save a schematic using Create''s <strong>Schematic and Quill</strong>.</li>
<li>The mod uploads the <code>.nbt</code> file to createmod.com automatically.</li>
<li>A clickable link appears in chat &mdash; visit it to claim and publish.</li>
</ol>
<p>The mod is client-side only, so it does not need to be installed on the server. If you prefer to upload manually, follow the steps below.</p>

<hr>

<h2>Manual Upload</h2>

<h3>Before You Upload</h3>
<p>To share your Create mod builds with the community, you need a <strong>.nbt schematic file</strong> exported from Minecraft. Make sure you have:</p>
<ul>
<li>A Minecraft world with the <a href="https://modrinth.com/mod/create" target="_blank">Create mod</a> installed.</li>
<li>A schematic of your build (created with Create''s Schematic and Quill item).</li>
<li>An account on CreateMod.com (register or log in first).</li>
</ul>

<h3>Step 1: Export Your Schematic</h3>
<p>In Minecraft with the Create mod:</p>
<ol>
<li>Craft a <strong>Schematic and Quill</strong> (empty schematic + feather in a crafting table).</li>
<li>Right-click to set the <strong>first corner</strong> of your build.</li>
<li>Right-click again to set the <strong>second corner</strong>. This defines the bounding box.</li>
<li>Confirm the selection by right-clicking a third time. A <code>.nbt</code> file is saved to your <code>.minecraft/schematics/</code> folder.</li>
</ol>
<p>You can find your schematics folder at:</p>
<ul>
<li><strong>Windows:</strong> <code>%appdata%\.minecraft\schematics\</code></li>
<li><strong>macOS:</strong> <code>~/Library/Application Support/minecraft/schematics/</code></li>
<li><strong>Linux:</strong> <code>~/.minecraft/schematics/</code></li>
</ul>

<h3>Step 2: Go to the Upload Page</h3>
<p>Click the <strong>Upload</strong> button in the site header, or navigate directly to <a href="/upload">/upload</a>. You must be logged in.</p>

<h3>Step 3: Upload Your .nbt File</h3>
<ol>
<li>Click <strong>Choose File</strong> and select your <code>.nbt</code> schematic file.</li>
<li>The file will be uploaded and analyzed automatically. Block counts, dimensions, and required mods are extracted from the NBT data.</li>
<li>You will be taken to a preview page where you can review the details.</li>
</ol>

<h3>Step 4: Add Details</h3>
<p>On the preview page you can:</p>
<ul>
<li>Add a <strong>title</strong> that describes your build (e.g., "Compact Brass Smelter").</li>
<li>Write a <strong>description</strong> explaining what the build does, how it works, and any tips for using it.</li>
<li>Add <strong>additional files</strong> like screenshots or alternative versions.</li>
<li>Choose a <strong>category</strong> and <strong>tags</strong> so others can find your schematic.</li>
</ul>

<h3>Step 5: Publish</h3>
<p>When you are happy with the details, click <strong>Publish</strong>. Your schematic will go through a brief moderation check and then appear on the site for the community to download.</p>

<h2>Tips for a Great Upload</h2>
<ul>
<li><strong>Use a descriptive title</strong>. "Automated Iron Farm" is better than "My Build 3".</li>
<li><strong>Write a clear description</strong>. Explain what the build does, the required mods/addons, and any setup instructions.</li>
<li><strong>Pick the right category</strong>. This helps users browsing by category find your work.</li>
<li><strong>Add clear screenshots</strong>. Viewers should be able to tell what the build is at a glance. Take screenshots in creative mode during daytime with the HUD hidden (press F1) for the best results.</li>
<li><strong>Keep file sizes reasonable</strong>. Trim your schematic selection box tightly around the build to avoid capturing empty space.</li>
</ul>'
WHERE id = 'guide_upload_001';
