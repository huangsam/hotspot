-- Version 6 Down: Revert magnitude metrics to INT (Postgres)

ALTER TABLE hotspot_file_scores_metrics 
    ALTER COLUMN total_commits TYPE INTEGER USING total_commits::INTEGER,
    ALTER COLUMN total_churn TYPE INTEGER USING total_churn::INTEGER,
    ALTER COLUMN contributor_count TYPE INTEGER USING contributor_count::INTEGER,
    ALTER COLUMN lines_added TYPE INTEGER USING lines_added::INTEGER,
    ALTER COLUMN lines_deleted TYPE INTEGER USING lines_deleted::INTEGER,
    ALTER COLUMN recent_commits TYPE INTEGER USING recent_commits::INTEGER,
    ALTER COLUMN recent_churn TYPE INTEGER USING recent_churn::INTEGER,
    ALTER COLUMN recent_lines_added TYPE INTEGER USING recent_lines_added::INTEGER,
    ALTER COLUMN recent_lines_deleted TYPE INTEGER USING recent_lines_deleted::INTEGER,
    ALTER COLUMN recent_contributor_count TYPE INTEGER USING recent_contributor_count::INTEGER,
    ALTER COLUMN age_days TYPE INTEGER USING age_days::INTEGER;
