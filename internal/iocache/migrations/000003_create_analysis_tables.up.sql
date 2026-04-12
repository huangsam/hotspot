-- Version 3: Create analysis tables with URN support
-- Establishes the core tables for analysis tracking and history with repository URN identifier.

CREATE TABLE IF NOT EXISTS hotspot_analysis_runs (
    analysis_id INTEGER PRIMARY KEY AUTOINCREMENT,
    start_time TEXT NOT NULL,
    end_time TEXT,
    run_duration_ms INTEGER,
    total_files_analyzed INTEGER,
    config_params TEXT,
    urn TEXT
);

CREATE INDEX IF NOT EXISTS idx_runs_urn ON hotspot_analysis_runs(urn);

CREATE TABLE IF NOT EXISTS hotspot_file_scores_metrics (
    analysis_id INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    analysis_time TEXT NOT NULL,
    total_commits INTEGER NOT NULL,
    total_churn INTEGER NOT NULL,
    contributor_count INTEGER NOT NULL,
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
