---
title: "Download Any Format, Schematic Tools & a Ship Hull Generator"
date: 2026-07-19
slug: schematic-tools-update
excerpt: "Download schematics in six formats or as a ready-to-play world, convert and inspect files with new tools, edit schematics in the browser, and generate ship hulls."
image: "/assets/x/news/tools-update.webp"
---

We have been working on this update for a while. Most of it is about the schematic files themselves: downloading them in the format you need, converting them, checking what is inside them, and even editing them right in the browser.

## Download in the Format You Use

The download button on schematic pages now has a menu. Besides the original Create .nbt you can download WorldEdit .schem, Litematica .litematic, legacy .schematic, MineColonies .blueprint, and Building Gadgets .json. There is also a ready-to-play world option that drops the build into a superflat world, so you can walk around it in game without pasting anything yourself.

![The download menu on a schematic page with Create .nbt, WorldEdit .schem, Litematica .litematic, legacy .schematic, MineColonies .blueprint, Building Gadgets .json and a ready-to-play world](/assets/x/news/tools-download-formats.webp)

## Schematic Converter

The [converter](/tools/convert) handles the same formats as the download menu, 30 conversion pairs in total. Drop up to 20 files at once, up to 10 MB each. Blocks, block entities, and Create mod data carry over. Files are converted in memory and never stored, and you do not need an account.

![The schematic converter with its supported conversion pairs](/assets/x/news/tools-converter.webp)

## Safety Checker

People ask whether schematics are safe to download. The short answer is that they are data, not programs, and cannot run code on your computer. Once pasted into a world though, some blocks can act on the world or on players: command blocks, spawners, click commands. The [safety checker](/tools/safety-check) scans a file and shows you exactly what is inside before you paste it.

![The safety checker explaining what schematics can and cannot contain](/assets/x/news/tools-safety-checker.webp)

## NBT Viewer

The [NBT viewer](/tools/nbt-viewer) opens any NBT or schematic file in a browsable tree, with SNBT view, key search, and copy-path. Handy for debugging a build, or just for seeing how the format works under the hood.

![The NBT viewer showing a schematic file as a browsable tree](/assets/x/news/tools-nbt-viewer.webp)

## A Schematic Editor, First Version

The [editor](/tools/editor) loads a schematic in 3D and lets you crop it to a region, fill and hollow, rotate and mirror, expand the build area, and replace one block type with another, with undo and redo. This first version renders everything as solid blocks, so stairs and slabs look chunky in the preview, but the file you download keeps the real block states. There is more we want to do here.

## Ship Hulls

The [generators](/generators) section has a new [hull generator](/generators/hull). Pick a preset like tugboat, destroyer, speedboat, yacht, or canoe, or shape your own with sliders for length, beam, depth, flare, and tumblehome. The new lofted hull engine produces much smoother curves than the old approach, and you can download the result as a schematic and build from there.

![A wooden hull generated with the new lofted hull engine](/assets/x/news/tools-hull-generator.webp)

## Similar by Shape

Schematic pages now have a "Similar by shape" button that finds builds with similar geometry rather than similar tags. There is also a [standalone version](/tools/similar) where you drop your own file and see what on the site resembles it. Both got sort options so you can order the results by rating, views, or date.

## Smaller Changes

- The search page has a proper Search button, for those of us who do not like waiting for the automatic search to kick in
- The 3D viewer now renders stairs and open trapdoors in their correct orientation
- Buttons on the gold background switched to dark text, which is much easier to read
- Pages load faster: the text editor only loads on pages that have one, and our CSS and JavaScript got smaller
- There is a [DMCA page](/dmca) for copyright requests, linked in the footer

As always, if something converts wrong or breaks, tell us on [GitHub](https://github.com/uberswe/createmod/issues) or the Discord. The links are in the footer.
