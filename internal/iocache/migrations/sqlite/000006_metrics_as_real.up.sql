-- Version 6: Convert magnitude metrics from INTEGER to REAL (SQLite)
-- Note: SQLite allows storing REAL in INTEGER columns, but this migration unifies the schema.

CREATE TABLE hotspot_file_scores_metrics_new (
    analysis_id INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    analysis_time TEXT NOT NULL,
    total_commits REAL NOT NULL,
    total_churn REAL NOT NULL,
    lines_added REAL NOT NULL,
    lines_deleted REAL NOT NULL,
    contributor_count REAL NOT NULL,
    recent_commits REAL NOT NULL,
    recent_churn REAL NOT NULL,
    recent_lines_added REAL NOT NULL,
    recent_lines_deleted REAL NOT NULL,
    recent_contributor_count REAL NOT NULL,
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

INSERT INTO hotspot_file_scores_metrics_new 
SELECT * FROM hotspot_file_scores_metrics;

DROP TABLE hotspot_file_scores_metrics;

ALTER TABLE hotspot_file_scores_metrics_new RENAME TO hotspot_file_scores_metrics;
