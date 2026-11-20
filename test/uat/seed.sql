-- Seed both engines with a table the @slow scenarios propose changes against. The syntax here is
-- the intersection of MySQL 8 and MariaDB 11 on purpose.
CREATE TABLE users (
  id INT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(120) NOT NULL UNIQUE,
  note VARCHAR(50) NULL
);
INSERT INTO users (email, note) VALUES ('a@x', 'one'), ('b@x', NULL), ('c@x', 'three');

-- The rehearsal creates and drops its own throwaway database, so the account it connects as needs
-- the privilege a real operator would give it. Without this the rehearsal degrades to tier=static,
-- which is correct but not what these scenarios prove.
GRANT ALL PRIVILEGES ON *.* TO 'noxis'@'%';
