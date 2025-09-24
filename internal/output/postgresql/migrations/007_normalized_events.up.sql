BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS api.events_raw (
  id          varchar(64) NOT NULL,  -- tx id
  event_index bigint      NOT NULL,  -- 0-based within the tx
  data        jsonb       NOT NULL,  -- full event JSON
  PRIMARY KEY (id, event_index),
  FOREIGN KEY (id) REFERENCES api.transactions_raw(id) ON DELETE CASCADE
);

-- Normalized events: one row per attribute
CREATE TABLE IF NOT EXISTS api.events_main (
  id          varchar(64) NOT NULL,   -- tx id
  event_index bigint      NOT NULL,   -- 0-based
  attr_index  bigint      NOT NULL,   -- 0-based within the event
  event_type  text        NOT NULL,
  attr_key    text        NOT NULL,
  attr_value  text,
  msg_index   bigint,                 -- nullable; from 'msg_index' attribute if present
  PRIMARY KEY (id, event_index, attr_index),
  FOREIGN KEY (id, event_index) REFERENCES api.events_raw(id, event_index) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS events_main_type_idx          ON api.events_main (event_type);
CREATE INDEX IF NOT EXISTS events_main_msg_idx           ON api.events_main (msg_index);
CREATE INDEX IF NOT EXISTS events_main_attr_key_val_sha256_idx ON api.events_main (attr_key, digest(COALESCE(attr_value, ''), 'sha256'));
CREATE INDEX IF NOT EXISTS events_main_id_idx            ON api.events_main (id);

CREATE OR REPLACE FUNCTION api.extract_event_msg_index(ev jsonb)
RETURNS bigint
LANGUAGE sql
STABLE
AS $$
  SELECT NULLIF(a->>'value','')::bigint
  FROM jsonb_array_elements(ev->'attributes') a
  WHERE a->>'key' = 'msg_index'
  LIMIT 1
$$;

-- Insert raw event on new raw transaction insert
CREATE OR REPLACE FUNCTION api.update_events_raw()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  ev jsonb;
  ev_ord int;
BEGIN
  -- Rebuild all events for this tx id (safe for INSERT and UPDATE)
  DELETE FROM api.events_raw WHERE id = NEW.id;

  FOR ev, ev_ord IN
    SELECT e, (ord::int - 1)
    FROM jsonb_array_elements(NEW.data->'txResponse'->'events') WITH ORDINALITY AS t(e, ord)
  LOOP
    INSERT INTO api.events_raw (id, event_index, data)
    VALUES (NEW.id, ev_ord, ev);
  END LOOP;

  RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS new_transaction_events_raw ON api.transactions_raw;
CREATE TRIGGER new_transaction_events_raw
AFTER INSERT OR UPDATE OF data
ON api.transactions_raw
FOR EACH ROW
EXECUTE FUNCTION api.update_events_raw();

-- Insert normalized event attributes on new raw event insert
CREATE OR REPLACE FUNCTION api.update_event_main()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  a jsonb;
  a_ord int;
  msg_idx bigint;
  ev_type text;
BEGIN
  -- Get msg_index once per event
  msg_idx := api.extract_event_msg_index(NEW.data);
  ev_type := NEW.data->>'type';

  -- Rebuild attributes for this (id, event_index)
  DELETE FROM api.events_main
  WHERE id = NEW.id AND event_index = NEW.event_index;

  FOR a, a_ord IN
    SELECT attr, (ord::int - 1)
    FROM jsonb_array_elements(NEW.data->'attributes') WITH ORDINALITY AS t(attr, ord)
  LOOP
    INSERT INTO api.events_main (
      id, event_index, attr_index, event_type, attr_key, attr_value, msg_index
    ) VALUES (
      NEW.id,
      NEW.event_index,
      a_ord,
      ev_type,
      a->>'key',
      a->>'value',
      msg_idx
    );
  END LOOP;

  RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS new_event_update ON api.events_raw;
CREATE TRIGGER new_event_update
AFTER INSERT OR UPDATE OF data
ON api.events_raw
FOR EACH ROW
EXECUTE FUNCTION api.update_event_main();

-- Backfill events_raw from existing transactions_raw (triggers to events_main will fire)
TRUNCATE api.events_raw CASCADE;

INSERT INTO api.events_raw (id, event_index, data)
SELECT tr.id,
       (ord::int - 1) AS event_index,
       ev
FROM api.transactions_raw tr
CROSS JOIN LATERAL jsonb_array_elements(tr.data->'txResponse'->'events')
  WITH ORDINALITY AS t(ev, ord);

GRANT SELECT ON api.events_raw  TO web_anon;
GRANT SELECT ON api.events_main TO web_anon;

COMMIT;
