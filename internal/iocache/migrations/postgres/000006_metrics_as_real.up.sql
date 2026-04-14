-- Version 6: Convert magnitude metrics to DOUBLE PRECISION (Postgres)

ALTER TABLE hotspot_file_scores_metrics 
    ALTER COLUMN total_commits TYPE DOUBLE PRECISION,
    ALTER COLUMN total_churn TYPE DOUBLE PRECISION,
    ALTER COLUMN contributor_count TYPE DOUBLE PRECISION,
    ALTER COLUMN lines_added TYPE DOUBLE PRECISION,
    ALTER COLUMN lines_deleted TYPE DOUBLE PRECISION,
    ALTER COLUMN recent_commits TYPE DOUBLE PRECISION,
    ALTER COLUMN recent_churn TYPE DOUBLE PRECISION,
    ALTER COLUMN recent_lines_added TYPE DOUBLE PRECISION,
    ALTER COLUMN recent_lines_deleted TYPE DOUBLE PRECISION,
    ALTER COLUMN recent_contributor_count TYPE DOUBLE PRECISION,
    ALTER COLUMN age_days TYPE DOUBLE PRECISION;
