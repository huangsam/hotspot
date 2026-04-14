-- Version 6 Down: Revert magnitude metrics to INT (MySQL)

ALTER TABLE hotspot_file_scores_metrics 
    MODIFY COLUMN total_commits INT NOT NULL,
    MODIFY COLUMN total_churn INT NOT NULL,
    MODIFY COLUMN contributor_count INT NOT NULL,
    MODIFY COLUMN lines_added INT NOT NULL,
    MODIFY COLUMN lines_deleted INT NOT NULL,
    MODIFY COLUMN recent_commits INT NOT NULL,
    MODIFY COLUMN recent_churn INT NOT NULL,
    MODIFY COLUMN recent_lines_added INT NOT NULL,
    MODIFY COLUMN recent_lines_deleted INT NOT NULL,
    MODIFY COLUMN recent_contributor_count INT NOT NULL,
    MODIFY COLUMN age_days INT NOT NULL;
