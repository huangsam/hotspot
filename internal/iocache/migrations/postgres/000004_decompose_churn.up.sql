-- Version 4: Decompose churn into added and deleted lines (PostgreSQL)

ALTER TABLE hotspot_file_scores_metrics ADD COLUMN lines_added INTEGER NOT NULL DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN lines_deleted INTEGER NOT NULL DEFAULT 0;
