ALTER TABLE schema_migrations ADD COLUMN checksum text;
INSERT INTO schema_migrations(version) VALUES(6);
