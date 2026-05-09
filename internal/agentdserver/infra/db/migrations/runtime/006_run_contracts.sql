ALTER TABLE agent_runs
ADD COLUMN input_json TEXT;

ALTER TABLE agent_runs
ADD COLUMN contract_input_schema_digest TEXT;

ALTER TABLE agent_runs
ADD COLUMN contract_output_schema_digest TEXT;

ALTER TABLE agent_runs
ADD COLUMN provider_name TEXT;

ALTER TABLE agent_runs
ADD COLUMN provider_model TEXT;

ALTER TABLE agent_runs
ADD COLUMN result_format TEXT NOT NULL DEFAULT 'text';
