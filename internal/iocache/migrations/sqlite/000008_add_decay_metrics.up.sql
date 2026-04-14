ALTER TABLE hotspot_file_scores_metrics ADD COLUMN decayed_commits REAL DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN decayed_churn REAL DEFAULT 0;
