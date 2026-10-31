-- The demo schema the dev stack seeds for MySQL and MariaDB.
CREATE DATABASE IF NOT EXISTS shop;

CREATE TABLE IF NOT EXISTS shop.customers (
  id          int AUTO_INCREMENT PRIMARY KEY,
  email       varchar(255) NOT NULL UNIQUE,
  full_name   varchar(255) NOT NULL,
  created_at  timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shop.orders (
  id           int AUTO_INCREMENT PRIMARY KEY,
  customer_id  int NOT NULL,
  status       varchar(32) NOT NULL,
  total_cents  int NOT NULL,
  placed_at    timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES shop.customers (id)
);

INSERT IGNORE INTO shop.customers (email, full_name) VALUES
  ('ada@example.com', 'Ada Lovelace'),
  ('grace@example.com', 'Grace Hopper');

INSERT INTO shop.orders (customer_id, status, total_cents)
SELECT c.id, 'paid', 4200 FROM shop.customers c
WHERE c.email = 'ada@example.com'
  AND NOT EXISTS (SELECT 1 FROM shop.orders o WHERE o.customer_id = c.id);
