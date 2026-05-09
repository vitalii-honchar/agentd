ALTER TABLE agents
ADD COLUMN contract_input_schema_raw TEXT;

ALTER TABLE agents
ADD COLUMN contract_output_schema_raw TEXT;

ALTER TABLE agents
ADD COLUMN contract_input_schema_digest TEXT;

ALTER TABLE agents
ADD COLUMN contract_output_schema_digest TEXT;

ALTER TABLE agent_revisions
ADD COLUMN contract_input_schema_raw TEXT;

ALTER TABLE agent_revisions
ADD COLUMN contract_output_schema_raw TEXT;

ALTER TABLE agent_revisions
ADD COLUMN contract_input_schema_digest TEXT;

ALTER TABLE agent_revisions
ADD COLUMN contract_output_schema_digest TEXT;

ALTER TABLE agent_revisions
ADD COLUMN contract_digest TEXT;
