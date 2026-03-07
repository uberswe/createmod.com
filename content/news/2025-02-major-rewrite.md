---
title: "Major Website Rewrite & Ownership Change"
date: 2025-02-15
slug: major-website-rewrite
excerpt: "CreateMod.com has undergone a full rewrite and is no longer running on Wordpress."
---

CreateMod.com has undergone a full rewrite and is no longer running on Wordpress as it had been doing since 2022.

This website was created by Slaser360 and during June 2024 the ownership was handed over to Uberswe

Below is a list of changes that have been done since June.

- Removed the majority of ads and all Google Adsense code
- Changed search to use in-memory [Bleve search](https://blevesearch.com/)
- Added ability to sort by ratings, views and more
- Complete redesign and new logo
- Removed lowest rated and most rated list from main page
- Added a trending list to the main page
- Fixed a bug allowing the same user to submit duplicate ratings once per day
- Cleaned up many bad schematics (will keep doing this)
- Compressed and optimized all images to reduce page size
- Added an Explore page for finding random schematics
- Made it possible to delete and edit schematics
- Max file size increased to 25mb for schematic files and images
- Made CreateMod.com open source, code can be found on [https://github.com/uberswe/createmod.com](https://github.com/uberswe/createmod.com)
- And so much more!

The current site uses Go for the backend and mostly vanilla JS with Bootstrap 5 for the frontend. If you would like to help make the site better please have a look at the [GitHub repository](https://github.com/uberswe/createmod.com) and open an issue or a pull request.

I hope the CreateMod.com will be a better experience for everyone going forward and I believe there are many interesting features that could further improve the overall experience.

### Site Traffic

Over time the site has received more and more popular and struggled with 100 concurrent users as I took it over. I spent a lot of time optimizing the old Wordpress site and ensuring that it has enough server capacity to serve traffic without crashes.

One issue was the incredible amount of data that is transferred from CreateMod.com every day. To help reduce the load I made use of [Cloudflare](https://www.cloudflare.com/). Below you can see a recent graph of the traffic CreateMod.com receives.

![CreateMod.com Traffic Jan 10 to Feb 9 2025](/assets/x/news/create-mod-stats-jan-feb-2025.png)
