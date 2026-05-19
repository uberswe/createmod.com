-- Remove tags whose name looks like a UUID (8-4-4-4-12 hex pattern)
-- and clean up their junction table entries.
DELETE FROM schematics_tags
WHERE tag_id IN (
    SELECT id FROM schematic_tags
    WHERE name ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
);

DELETE FROM schematic_tags
WHERE name ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$';
