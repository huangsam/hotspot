-- Version 3: Create analysis tables with URN support (MySQL)

CREATE TABLE IF NOT EXISTS hotspot_analysis_runs (
    analysis_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    start_time DATETIME(6) NOT NULL,
    end_time DATETIME(6),
    run_duration_ms INT,
    total_files_analyzed INT,
    config_params TEXT,
    urn VARCHAR(255)
);

CREATE INDEX idx_runs_urn ON hotspot_analysis_runs(urn);

CREATE TABLE IF NOT EXISTS hotspot_file_scores_metrics (
    analysis_id BIGINT NOT NULL,
    file_path VARCHAR(512) NOT NULL,
    analysis_time DATETIME(6) NOT NULL,
    total_commits INT NOT NULL,
    total_churn INT NOT NULL,
    contributor_count INT NOT NULL,
    age_days DOUBLE NOT NULL,
    gini_coefficient DOUBLE NOT NULL,
    file_owner VARCHAR(100),
    score_hot DOUBLE NOT NULL,
    score_risk DOUBLE NOT NULL,
    score_complexity DOUBLE NOT NULL,
    score_label VARCHAR(50) NOT NULL,
    PRIMARY KEY (analysis_id, file_path)
);
