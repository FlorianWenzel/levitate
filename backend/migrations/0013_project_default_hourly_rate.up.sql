ALTER TABLE projects
    ADD COLUMN default_hourly_rate numeric(12, 3) NOT NULL DEFAULT 0;
