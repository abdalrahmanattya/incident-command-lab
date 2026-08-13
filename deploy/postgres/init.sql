CREATE TABLE IF NOT EXISTS outbox_events (
  id uuid PRIMARY KEY, aggregate_id text NOT NULL, event_type text NOT NULL,
  payload jsonb NOT NULL, status text NOT NULL CHECK (status IN ('PENDING','RETRY','DELIVERED','DLQ')),
  attempts integer NOT NULL DEFAULT 0, available_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(), last_error text
);
CREATE INDEX IF NOT EXISTS outbox_ready_idx ON outbox_events(status, available_at);
CREATE TABLE IF NOT EXISTS idempotency_keys (key text PRIMARY KEY, response jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS products (id text PRIMARY KEY, name text NOT NULL, price_cents integer NOT NULL CHECK(price_cents > 0), stock integer NOT NULL CHECK(stock >= 0));
CREATE TABLE IF NOT EXISTS reservations (id text PRIMARY KEY, key text UNIQUE NOT NULL, product_id text REFERENCES products(id), customer_id text NOT NULL, status text NOT NULL, quantity integer NOT NULL, total_cents integer NOT NULL, release text NOT NULL, failure text, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL);
INSERT INTO products(id,name,price_cents,stock) VALUES ('concert','Concert ticket',2500,100),('workshop','Reliability workshop',7500,20) ON CONFLICT (id) DO NOTHING;
CREATE TABLE IF NOT EXISTS processed_events (event_id uuid PRIMARY KEY, consumer text NOT NULL, processed_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS payment_outcomes (event_id uuid PRIMARY KEY, reservation_id text NOT NULL, status text NOT NULL, reason text, processed_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS notification_outcomes (event_id uuid PRIMARY KEY, reservation_id text NOT NULL, status text NOT NULL, processed_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS fault_flags (name text PRIMARY KEY, enabled boolean NOT NULL, updated_at timestamptz NOT NULL DEFAULT now());
