-- Float-parity project tags. Float's Project API exposes a `tags[]` array of
-- free-form strings used to categorize projects (see
-- https://developer.float.com/swagger-api-v3.yaml, projects path). We persist
-- them as a non-null text array so the JSON response always returns [] for
-- projects without tags.
ALTER TABLE projects
    ADD COLUMN tags text[] NOT NULL DEFAULT '{}';
