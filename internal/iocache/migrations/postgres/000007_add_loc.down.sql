-- Version 7 Down: Remove lines_of_code column (Postgres)

ALTER TABLE hotspot_file_scores_metrics DROP COLUMN lines_of_code;
