ALTER TABLE prepared_features ADD COLUMN calendar_flags jsonb NOT NULL DEFAULT '{}';
UPDATE prepared_features SET calendar_flags=jsonb_strip_nulls(jsonb_build_object(
 'is_holiday',features->'is_holiday','is_weekend',features->'is_weekend'));
INSERT INTO schema_migrations(version) VALUES(5);
