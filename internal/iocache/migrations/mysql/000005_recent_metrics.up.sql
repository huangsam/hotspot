ALTER TABLE hotspot_file_scores_metrics 
ADD COLUMN recent_commits INT NOT NULL DEFAULT 0,
ADD COLUMN recent_churn INT NOT NULL DEFAULT 0,
ADD COLUMN recent_lines_added INT NOT NULL DEFAULT 0,
ADD COLUMN recent_lines_deleted INT NOT NULL DEFAULT 0,
ADD COLUMN recent_contributor_count INT NOT NULL DEFAULT 0;
