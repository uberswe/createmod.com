# Legacy block flattening data

`legacy_blocks.json` maps pre-1.13 numeric block IDs (`"id:meta"`) to modern
namespaced blockstate strings — the factual correspondence defined by
Mojang's 1.13 "flattening" (DataFixerUpper).

Source: `data/pc/common/legacy.json` from PrismarineJS/minecraft-data
(https://github.com/PrismarineJS/minecraft-data), MIT-licensed per its
README.

Local correction: minecraft-data ships eight stairs entries with
`shape=outer_right`; DataFixerUpper flattens all pre-1.13 stairs to
`shape=straight` (the shape property is recomputed on placement in-game),
so those entries are corrected here.
