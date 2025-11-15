-- migrate:up

-- Create UUID v7 generation function
-- UUID v7 is time-sortable and improves index performance
CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS UUID
AS $$
DECLARE
  unix_ts_ms BIGINT;
  uuid_bytes BYTEA;
BEGIN
  unix_ts_ms = (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;
  uuid_bytes = SET_BYTE(SET_BYTE('\x00000000000000000000000000000000'::BYTEA, 0, (unix_ts_ms >> 40)::BIT(8)::INTEGER),
                        1, (unix_ts_ms >> 32)::BIT(8)::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 2, (unix_ts_ms >> 24)::BIT(8)::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 3, (unix_ts_ms >> 16)::BIT(8)::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 4, (unix_ts_ms >> 8)::BIT(8)::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 5, unix_ts_ms::BIT(8)::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 6, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 7, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 8, (floor(random() * 255))::INTEGER | (2 << 6)); -- Set version and variant
  uuid_bytes = SET_BYTE(uuid_bytes, 9, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 10, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 11, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 12, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 13, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 14, (floor(random() * 255))::INTEGER);
  uuid_bytes = SET_BYTE(uuid_bytes, 15, (floor(random() * 255))::INTEGER);

  RETURN ENCODE(uuid_bytes, 'hex')::UUID;
END
$$
LANGUAGE PLPGSQL
VOLATILE;

-- migrate:down
DROP FUNCTION IF EXISTS uuid_generate_v7();
