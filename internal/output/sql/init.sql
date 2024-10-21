-- Create the schema if it doesn't exist
CREATE SCHEMA IF NOT EXISTS api;

-- Create the tables if they don't exist
    CREATE TABLE IF NOT EXISTS api.blocks (
        id SERIAL PRIMARY KEY,
        data JSONB NOT NULL
    );
    CREATE TABLE IF NOT EXISTS api.transactions (
        id VARCHAR(64) PRIMARY KEY,
        data JSONB NOT NULL
    );

-- Create a role for anonymous web access if it doesn't exist
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'web_anon') THEN
    CREATE ROLE web_anon NOLOGIN;
  END IF;
END
$$;

-- Grant access to the web_anon role. Will succeed even if the role already has access.
GRANT USAGE ON SCHEMA api TO web_anon;
GRANT SELECT ON api.blocks TO web_anon;
GRANT SELECT ON api.transactions TO web_anon;
