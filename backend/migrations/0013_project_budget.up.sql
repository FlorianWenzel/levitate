ALTER TABLE projects
    ADD COLUMN budget_type     smallint,
    ADD COLUMN budget_total    numeric,
    ADD COLUMN budget_priority smallint;
