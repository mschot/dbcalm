-- Incremental dataset for E2E tests
-- This data is added after the full backup to test incremental backups

INSERT INTO users (id, username, email, created_at) VALUES
(6, 'frank', 'frank@example.com', '2025-01-03 10:00:00'),
(7, 'grace', 'grace@example.com', '2025-01-03 11:00:00'),
(8, 'henry', 'henry@example.com', '2025-01-03 12:00:00');

INSERT INTO orders (id, user_id, order_number, order_date, total, status) VALUES
(6, 5, 'ORD-006', '2025-01-03 10:00:00', 125.75, 'completed'),
(7, 6, 'ORD-007', '2025-01-03 11:00:00', 89.99, 'pending');

-- Update some existing records to test incremental changes
UPDATE orders SET status = 'completed' WHERE id = 4;
UPDATE users SET email = 'charlie.updated@example.com' WHERE id = 3;
