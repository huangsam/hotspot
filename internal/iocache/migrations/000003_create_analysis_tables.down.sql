-- Version 3 down: Drop analysis tables
-- Note: This is a destructive operation. Consider backup before rollback.

DROP TABLE IF EXISTS hotspot_file_scores_metrics;
DROP TABLE IF EXISTS hotspot_analysis_runs;
DROP INDEX IF EXISTS idx_runs_urn;
