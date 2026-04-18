-- Version 9: Intelligence Layer - Reasoning, ROI, and Recency Signals (MySQL)
ALTER TABLE hotspot_file_scores_metrics
ADD COLUMN reasoning JSON,
ADD COLUMN score_roi DOUBLE DEFAULT 0,
ADD COLUMN recency_signal DOUBLE DEFAULT 0,
ADD COLUMN recency_threshold_low DOUBLE DEFAULT 0,
ADD COLUMN recency_threshold_high DOUBLE DEFAULT 0;
