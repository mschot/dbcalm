-- Initial dataset for E2E tests
-- This data represents the state after first backup

INSERT INTO users (id, username, email, created_at) VALUES
(1, 'alice', 'alice@example.com', '2025-01-01 10:00:00'),
(2, 'bob', 'bob@example.com', '2025-01-01 11:00:00'),
(3, 'charlie', 'charlie@example.com', '2025-01-01 12:00:00'),
(4, 'diana', 'diana@example.com', '2025-01-01 13:00:00'),
(5, 'eve', 'eve@example.com', '2025-01-01 14:00:00');

INSERT INTO orders (id, user_id, order_number, order_date, total, status) VALUES
(1, 1, 'ORD-001', '2025-01-02 10:00:00', 99.99, 'completed'),
(2, 1, 'ORD-002', '2025-01-02 11:00:00', 149.50, 'completed'),
(3, 2, 'ORD-003', '2025-01-02 12:00:00', 75.25, 'completed'),
(4, 3, 'ORD-004', '2025-01-02 13:00:00', 200.00, 'pending'),
(5, 4, 'ORD-005', '2025-01-02 14:00:00', 50.00, 'completed');
