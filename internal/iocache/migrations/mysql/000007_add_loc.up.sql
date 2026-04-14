-- Version 7: Add lines_of_code metric to File Scores and Metrics (MySQL)
-- This facilitates snapshotting complexity/LOC for historical analysis.

ALTER TABLE hotspot_file_scores_metrics ADD COLUMN lines_of_code DOUBLE PRECISION NOT NULL DEFAULT 0;
