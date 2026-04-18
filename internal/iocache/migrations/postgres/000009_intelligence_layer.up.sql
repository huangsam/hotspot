-- Version 9: Intelligence Layer - Reasoning, ROI, and Recency Signals (PostgreSQL)
ALTER TABLE hotspot_file_scores_metrics
ADD COLUMN reasoning JSONB,
ADD COLUMN score_roi DOUBLE PRECISION DEFAULT 0,
ADD COLUMN recency_signal DOUBLE PRECISION DEFAULT 0,
ADD COLUMN recency_threshold_low DOUBLE PRECISION DEFAULT 0,
ADD COLUMN recency_threshold_high DOUBLE PRECISION DEFAULT 0;
