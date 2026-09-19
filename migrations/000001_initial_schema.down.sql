-- Reverses 000001_initial_schema.up.sql, in dependency order (drop
-- tables that reference others before the tables they reference).

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS job_attempts;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS stages;
DROP TABLE IF EXISTS pipelines;
DROP TABLE IF EXISTS projects;
