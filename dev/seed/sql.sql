-- The demo schema the dev stack seeds: small enough to read, related enough to draw.
CREATE SCHEMA IF NOT EXISTS shop;

CREATE TABLE IF NOT EXISTS shop.customers (
  id          serial PRIMARY KEY,
  email       text NOT NULL UNIQUE,
  full_name   text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shop.orders (
  id           serial PRIMARY KEY,
  customer_id  integer NOT NULL REFERENCES shop.customers (id),
  status       text NOT NULL,
  total_cents  integer NOT NULL,
  placed_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shop.order_items (
  id        serial PRIMARY KEY,
  order_id  integer NOT NULL REFERENCES shop.orders (id),
  sku       text NOT NULL,
  qty       integer NOT NULL,
  unit_cents integer NOT NULL
);

INSERT INTO shop.customers (email, full_name) VALUES
  ('ada@example.com', 'Ada Lovelace'),
  ('grace@example.com', 'Grace Hopper'),
  ('alan@example.com', 'Alan Turing')
ON CONFLICT (email) DO NOTHING;

INSERT INTO shop.orders (customer_id, status, total_cents)
SELECT c.id, s.status, s.total
FROM (VALUES ('ada@example.com', 'paid', 4200), ('grace@example.com', 'pending', 1999)) AS s(email, status, total)
JOIN shop.customers c ON c.email = s.email
WHERE NOT EXISTS (SELECT 1 FROM shop.orders o WHERE o.customer_id = c.id AND o.total_cents = s.total);

INSERT INTO shop.order_items (order_id, sku, qty, unit_cents)
SELECT o.id, 'SKU-1', 2, 2100
FROM shop.orders o
WHERE o.total_cents = 4200 AND NOT EXISTS (SELECT 1 FROM shop.order_items i WHERE i.order_id = o.id);
