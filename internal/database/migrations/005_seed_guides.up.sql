-- Seed default guides for new installations.
-- Uses ON CONFLICT DO NOTHING so re-running migrations won't duplicate rows.

INSERT INTO guides (id, author_id, title, description, content, slug, upload_link)
VALUES (
    'guide_upload_001',
    NULL,
    'How to Upload a Schematic',
    'A step-by-step guide on uploading your Create mod schematics to CreateMod.com.',
    '<h2>Before You Upload</h2>
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
<li>Right-click again to set the <strong>second corner</strong> — this defines the bounding box.</li>
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
<li>The file will be uploaded and analyzed automatically — block counts, dimensions, and required mods are extracted from the NBT data.</li>
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
<li><strong>Use a descriptive title</strong> — "Automated Iron Farm" is better than "My Build 3".</li>
<li><strong>Write a clear description</strong> — explain what the build does, the required mods/addons, and any setup instructions.</li>
<li><strong>Pick the right category</strong> — this helps users browsing by category find your work.</li>
<li><strong>Add screenshots</strong> — a picture is worth a thousand words. Show the build in action!</li>
<li><strong>Keep file sizes reasonable</strong> — trim your schematic selection box tightly around the build to avoid capturing empty space.</li>
</ul>',
    'how-to-upload-a-schematic',
    ''
) ON CONFLICT (id) DO NOTHING;

INSERT INTO guides (id, author_id, title, description, content, slug, upload_link)
VALUES (
    'guide_install_001',
    NULL,
    'How to Use a Schematic',
    'Learn how to download and use schematics from CreateMod.com in your Minecraft world.',
    '<h2>What You Need</h2>
<ul>
<li>Minecraft with the <a href="https://modrinth.com/mod/create" target="_blank">Create mod</a> installed.</li>
<li>A downloaded <code>.nbt</code> schematic file from CreateMod.com.</li>
</ul>

<h2>Step 1: Download the Schematic</h2>
<p>Find a schematic you like on the site, then click the <strong>Download</strong> button on its detail page. The <code>.nbt</code> file will be saved to your computer.</p>

<h2>Step 2: Copy to the Schematics Folder</h2>
<p>Move or copy the downloaded <code>.nbt</code> file into your Minecraft schematics folder:</p>
<ul>
<li><strong>Windows:</strong> <code>%appdata%\.minecraft\schematics\</code></li>
<li><strong>macOS:</strong> <code>~/Library/Application Support/minecraft/schematics/</code></li>
<li><strong>Linux:</strong> <code>~/.minecraft/schematics/</code></li>
</ul>
<p>If the <code>schematics</code> folder does not exist, create it manually.</p>

<h2>Step 3: Place the Schematic in Your World</h2>
<ol>
<li>Open your Minecraft world.</li>
<li>Craft or obtain a <strong>Schematic and Quill</strong>, then right-click it in an empty area — this isn''t strictly needed but helps you learn the tool.</li>
<li>Craft an <strong>Empty Schematic</strong> (paper in a crafting table) and open the <strong>Schematic Table</strong>.</li>
<li>Place your <strong>Empty Schematic</strong> in the Schematic Table, and you should see a list of available schematics including the one you downloaded.</li>
<li>Select the schematic and click <strong>Upload Schematic</strong> to write it to the empty schematic item.</li>
<li>Place the filled schematic into a <strong>Schematicannon</strong> along with building materials (blocks).</li>
<li>Use the <strong>Schematicannon</strong> to build the structure automatically, or use the <strong>Schematic</strong> item directly to get a holographic preview and place blocks manually.</li>
</ol>

<h2>Alternative: Direct Placement with Schematic Item</h2>
<p>You can also hold a filled Schematic item and right-click to get a translucent preview of the build. Use the on-screen controls to position, rotate, and mirror the schematic before placing blocks by hand or with a Schematicannon.</p>

<h2>Troubleshooting</h2>
<ul>
<li><strong>Schematic not appearing?</strong> Make sure the file is in the correct <code>schematics</code> folder and has a <code>.nbt</code> extension.</li>
<li><strong>Missing blocks?</strong> The schematic may use blocks from addons (e.g., Create: Steam ''n'' Rails, Create: Crafts &amp; Additions). Install the required mods listed on the schematic''s page.</li>
<li><strong>Build looks wrong?</strong> Check that you are using the same Minecraft and Create mod version as the schematic author.</li>
</ul>',
    'how-to-use-a-schematic',
    ''
) ON CONFLICT (id) DO NOTHING;

INSERT INTO guides (id, author_id, title, description, content, slug, upload_link)
VALUES (
    'guide_getting_001',
    NULL,
    'Getting Started with the Create Mod',
    'An introduction to the Create mod for Minecraft — what it is, how to install it, and first steps.',
    '<h2>What Is the Create Mod?</h2>
<p>The <a href="https://modrinth.com/mod/create" target="_blank">Create mod</a> is a Minecraft mod focused on building mechanical contraptions, automation, and aesthetic machinery. Unlike many tech mods, Create emphasizes <strong>visual, physical machines</strong> — you can watch gears turn, belts move, and pistons push in real time.</p>
<p>Key features include:</p>
<ul>
<li><strong>Rotational power</strong> — waterwheels, windmills, and hand cranks generate Stress Units (SU) that drive machines.</li>
<li><strong>Mechanical components</strong> — shafts, gearboxes, belts, and clutches transfer and control rotation.</li>
<li><strong>Automation</strong> — mechanical crafters, deployers, and funnels let you automate almost any recipe.</li>
<li><strong>Logistics</strong> — belts, chutes, and smart funnels move items around your factory.</li>
<li><strong>Trains</strong> — build and ride custom trains across your world with the Create train system.</li>
<li><strong>Schematics</strong> — save, share, and deploy builds using the schematic system (that is what this site is for!).</li>
</ul>

<h2>Installing the Create Mod</h2>
<h3>Using a Mod Launcher (Recommended)</h3>
<p>The easiest way is to use a launcher that supports Fabric or Forge/NeoForge:</p>
<ol>
<li>Install <a href="https://modrinth.com/app" target="_blank">Modrinth App</a>, <a href="https://prismlauncher.org/" target="_blank">Prism Launcher</a>, or <a href="https://www.curseforge.com/download/app" target="_blank">CurseForge App</a>.</li>
<li>Search for <strong>"Create"</strong> in the mod browser.</li>
<li>Click Install — the launcher handles dependencies automatically.</li>
</ol>
<h3>Manual Installation</h3>
<ol>
<li>Install <a href="https://fabricmc.net/" target="_blank">Fabric Loader</a> or <a href="https://neoforged.net/" target="_blank">NeoForge</a> for your Minecraft version.</li>
<li>Download Create from <a href="https://modrinth.com/mod/create" target="_blank">Modrinth</a> or <a href="https://www.curseforge.com/minecraft/mc-mods/create" target="_blank">CurseForge</a>.</li>
<li>Place the <code>.jar</code> file in your <code>.minecraft/mods/</code> folder.</li>
<li>Install any required dependencies (the download page will list them).</li>
<li>Launch Minecraft with the modded profile.</li>
</ol>

<h2>Your First Contraption</h2>
<p>A great starting point is a <strong>waterwheel setup</strong>:</p>
<ol>
<li>Find or create a water source with flowing water.</li>
<li>Craft a <strong>Water Wheel</strong> and place it in the flowing water — it starts generating rotational force (SU).</li>
<li>Attach a <strong>Shaft</strong> to the water wheel and extend it to where you want to use the power.</li>
<li>Connect a <strong>Mechanical Press</strong> or <strong>Millstone</strong> to process items.</li>
<li>Use <strong>Belts</strong> and <strong>Funnels</strong> to feed items in and collect the output.</li>
</ol>
<p>Experiment! The Create mod rewards creative problem-solving and there is no single "right" way to build.</p>

<h2>Learning More</h2>
<ul>
<li>The in-game <strong>Ponder</strong> system (hold W while hovering over a Create item) shows animated tutorials for every component.</li>
<li>Browse schematics on this site to see how other players build their contraptions.</li>
<li>Check out the other guides in our <a href="/guides">Guides section</a> for more detailed tutorials.</li>
</ul>',
    'getting-started-with-create-mod',
    ''
) ON CONFLICT (id) DO NOTHING;
