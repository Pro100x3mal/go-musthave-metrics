CREATE TABLE metrics
(
    "id"    VARCHAR(255) NOT NULL PRIMARY KEY,
    "mtype" VARCHAR(255) NOT NULL,
    "delta" BIGINT,
    "value" DOUBLE PRECISION
);
CREATE INDEX idx_metrics_mtype ON metrics (mtype);