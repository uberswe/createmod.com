-- Revert the upload guide to the original content (without the mod promo).
UPDATE guides
SET content = '<h2>Before You Upload</h2>
<p>To share your Create mod builds with the community, you need a <strong>.nbt schematic file</strong> exported from Minecraft. Make sure you have:</p>
<ul>
<li>A Minecraft world with the <a href="https://modrinth.com/mod/create" target="_blank">Create mod</a> installed.</li>
<li>A schematic of your build (created with Create''s Schematic and Quill item).</li>
<li>An account on CreateMod.com (register or log in first).</li>
</ul>

<h2>Step 1: Export Your Schematic</h2>
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

<h2>Step 2: Go to the Upload Page</h2>
<p>Click the <strong>Upload</strong> button in the site header, or navigate directly to <a href="/upload">/upload</a>. You must be logged in.</p>

<h2>Step 3: Upload Your .nbt File</h2>
<ol>
<li>Click <strong>Choose File</strong> and select your <code>.nbt</code> schematic file.</li>
<li>The file will be uploaded and analyzed automatically. Block counts, dimensions, and required mods are extracted from the NBT data.</li>
<li>You will be taken to a preview page where you can review the details.</li>
</ol>

<h2>Step 4: Add Details</h2>
<p>On the preview page you can:</p>
<ul>
<li>Add a <strong>title</strong> that describes your build (e.g., "Compact Brass Smelter").</li>
<li>Write a <strong>description</strong> explaining what the build does, how it works, and any tips for using it.</li>
<li>Add <strong>additional files</strong> like screenshots or alternative versions.</li>
<li>Choose a <strong>category</strong> and <strong>tags</strong> so others can find your schematic.</li>
</ul>

<h2>Step 5: Publish</h2>
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
