-- Version 4: Decompose churn into added and deleted lines (MySQL)

ALTER TABLE hotspot_file_scores_metrics ADD COLUMN lines_added INT NOT NULL DEFAULT 0;
ALTER TABLE hotspot_file_scores_metrics ADD COLUMN lines_deleted INT NOT NULL DEFAULT 0;
