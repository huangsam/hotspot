ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recent_commits INTEGER NOT NULL DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recent_churn INTEGER NOT NULL DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recent_lines_added INTEGER NOT NULL DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recent_lines_deleted INTEGER NOT NULL DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN recent_contributor_count INTEGER NOT NULL DEFAULT 0;
