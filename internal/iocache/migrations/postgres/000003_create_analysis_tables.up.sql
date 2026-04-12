-- Version 3: Create analysis tables with URN support (PostgreSQL)

CREATE TABLE IF NOT EXISTS hotspot_analysis_runs (
    analysis_id BIGSERIAL PRIMARY KEY,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    run_duration_ms INT,
    total_files_analyzed INT,
    config_params TEXT,
    urn TEXT
);

CREATE INDEX IF NOT EXISTS idx_runs_urn ON hotspot_analysis_runs(urn);

CREATE TABLE IF NOT EXISTS hotspot_file_scores_metrics (
    analysis_id BIGINT NOT NULL,
    file_path TEXT NOT NULL,
    analysis_time TIMESTAMPTZ NOT NULL,
    total_commits INT NOT NULL,
    total_churn INT NOT NULL,
    contributor_count INT NOT NULL,
    age_days DOUBLE PRECISION NOT NULL,
    gini_coefficient DOUBLE PRECISION NOT NULL,
    file_owner TEXT,
    score_hot DOUBLE PRECISION NOT NULL,
    score_risk DOUBLE PRECISION NOT NULL,
    score_complexity DOUBLE PRECISION NOT NULL,
    score_stale DOUBLE PRECISION NOT NULL,
    score_label TEXT NOT NULL,
    PRIMARY KEY (analysis_id, file_path)
);
