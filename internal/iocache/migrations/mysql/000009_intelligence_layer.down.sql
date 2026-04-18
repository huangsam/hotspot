-- Version 9: Remove Intelligence Layer (MySQL)
ALTER TABLE hotspot_file_scores_metrics
DROP COLUMN reasoning,
DROP COLUMN score_roi,
DROP COLUMN recency_signal,
DROP COLUMN recency_threshold_low,
DROP COLUMN recency_threshold_high;
