ALTER TABLE hotspot_file_scores_metrics ADD COLUMN decayed_commits DOUBLE PRECISION DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN decayed_churn DOUBLE PRECISION DEFAULT 0;
