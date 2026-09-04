CREATE TABLE gauge_metrics (
    name VARCHAR(255) PRIMARY KEY,
    value DOUBLE PRECISION NOT NULL
);

CREATE TABLE counter_metrics (
    name VARCHAR(255) PRIMARY KEY,
    value BIGINT NOT NULL
);