# Legacy block flattening data

`legacy_blocks.json` maps pre-1.13 numeric block IDs (`"id:meta"`) to modern
namespaced blockstate strings. It is the block section of WorldEdit's
`legacy.json` (https://github.com/EngineHub/WorldEdit,
worldedit-core/src/main/resources/com/sk89q/worldedit/world/registry/legacy.json).

The underlying mapping is factual — it records Mojang's own 1.13 "flattening"
correspondence — but the file originates from the WorldEdit project, which is
licensed GPL-3.0. If that provenance is a concern, regenerate this table from
Mojang's data generators or another source before shipping.
