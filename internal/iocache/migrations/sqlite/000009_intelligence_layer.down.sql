-- Version 9: Remove Intelligence Layer (SQLite)
ALTER TABLE hotspot_file_scores_metrics DROP COLUMN reasoning;
ALTER TABLE hotspot_file_scores_metrics DROP COLUMN score_roi;
ALTER TABLE hotspot_file_scores_metrics DROP COLUMN recency_signal;
ALTER TABLE hotspot_file_scores_metrics DROP COLUMN recency_threshold_low;
ALTER TABLE hotspot_file_scores_metrics DROP COLUMN recency_threshold_high;
