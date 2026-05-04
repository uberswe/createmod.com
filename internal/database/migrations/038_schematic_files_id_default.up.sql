ALTER TABLE schematic_files ALTER COLUMN id SET DEFAULT gen_random_uuid()::text;
