-- The seed the local compose stack gives the target it asks you to register: one table with a
-- primary key, a couple of columns worth completing, and enough rows that `SELECT` returns something.
CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  email VARCHAR(254) NOT NULL,
  note TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY email (email)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  total_cents INT NOT NULL,
  placed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY user_id (user_id)
) ENGINE = InnoDB;

INSERT INTO users (email, note) VALUES
  ('ada@example.com', 'first'),
  ('grace@example.com', 'second');

INSERT INTO orders (user_id, total_cents) VALUES (1, 1250), (2, 3400);
