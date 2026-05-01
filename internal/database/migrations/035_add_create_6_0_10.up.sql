INSERT INTO createmod_versions (id, version, created, updated)
VALUES ('create_6_0_10', '6.0.10', NOW(), NOW())
ON CONFLICT DO NOTHING;
