-- Add conservative upstream-account health alerts without changing schema.
-- Existing names make the migration idempotent across restored databases.

INSERT INTO ops_alert_rules (
    name, description, enabled, metric_type, operator, threshold,
    window_minutes, sustained_minutes, severity, notify_email, cooldown_minutes,
    created_at, updated_at
) VALUES (
    '上游账号异常',
    '当存在非临时不可调度的错误账号且持续 5 分钟时触发告警',
    true, 'account_error_count', '>', 0.0, 5, 5, 'P1', true, 20, NOW(), NOW()
) ON CONFLICT (name) DO NOTHING;

INSERT INTO ops_alert_rules (
    name, description, enabled, metric_type, operator, threshold,
    window_minutes, sustained_minutes, severity, notify_email, cooldown_minutes,
    created_at, updated_at
) VALUES (
    '上游账号异常比例过高',
    '当错误账号比例超过 25% 且持续 5 分钟时触发告警',
    true, 'account_error_ratio', '>', 25.0, 5, 5, 'P1', true, 20, NOW(), NOW()
) ON CONFLICT (name) DO NOTHING;

INSERT INTO ops_alert_rules (
    name, description, enabled, metric_type, operator, threshold,
    window_minutes, sustained_minutes, severity, notify_email, cooldown_minutes,
    created_at, updated_at
) VALUES (
    '上游账号持续限流',
    '当存在受限账号且持续 10 分钟时触发告警',
    true, 'account_rate_limited_count', '>', 0.0, 5, 10, 'P2', true, 30, NOW(), NOW()
) ON CONFLICT (name) DO NOTHING;
