-- Version 4: Decompose churn into added and deleted lines (MySQL)

ALTER TABLE hotspot_file_scores_metrics DROP COLUMN lines_added;
ALTER TABLE hotspot_file_scores_metrics DROP COLUMN lines_deleted;
