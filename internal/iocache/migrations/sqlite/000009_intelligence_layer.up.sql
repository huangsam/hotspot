-- Version 9: Intelligence Layer - Reasoning, ROI, and Recency Signals (SQLite)
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN reasoning TEXT;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN score_roi DOUBLE PRECISION DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recency_signal DOUBLE PRECISION DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recency_threshold_low DOUBLE PRECISION DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recency_threshold_high DOUBLE PRECISION DEFAULT 0;
