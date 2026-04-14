-- Version 6 Down: Revert magnitude metrics from REAL to INTEGER (SQLite)

CREATE TABLE hotspot_file_scores_metrics_down (
    analysis_id INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    analysis_time TEXT NOT NULL,
    total_commits INTEGER NOT NULL,
    total_churn INTEGER NOT NULL,
    lines_added INTEGER NOT NULL,
    lines_deleted INTEGER NOT NULL,
    contributor_count INTEGER NOT NULL,
    recent_commits INTEGER NOT NULL,
    recent_churn INTEGER NOT NULL,
    recent_lines_added INTEGER NOT NULL,
    recent_lines_deleted INTEGER NOT NULL,
    recent_contributor_count INTEGER NOT NULL,
    age_days REAL NOT NULL,
    gini_coefficient REAL NOT NULL,
    file_owner TEXT,
    score_hot REAL NOT NULL,
    score_risk REAL NOT NULL,
    score_complexity REAL NOT NULL,
    score_stale REAL NOT NULL,
    score_label TEXT NOT NULL,
    PRIMARY KEY (analysis_id, file_path)
);

-- Truncate fractional values when reverting to INTEGER
INSERT INTO hotspot_file_scores_metrics_down 
SELECT 
    analysis_id,
    file_path,
    analysis_time,
    CAST(total_commits AS INTEGER),
    CAST(total_churn AS INTEGER),
    CAST(lines_added AS INTEGER),
    CAST(lines_deleted AS INTEGER),
    CAST(contributor_count AS INTEGER),
    CAST(recent_commits AS INTEGER),
    CAST(recent_churn AS INTEGER),
    CAST(recent_lines_added AS INTEGER),
    CAST(recent_lines_deleted AS INTEGER),
    CAST(recent_contributor_count AS INTEGER),
    age_days,
    gini_coefficient,
    file_owner,
    score_hot,
    score_risk,
    score_complexity,
    score_stale,
    score_label
FROM hotspot_file_scores_metrics;

DROP TABLE hotspot_file_scores_metrics;

ALTER TABLE hotspot_file_scores_metrics_down RENAME TO hotspot_file_scores_metrics;
