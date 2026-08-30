-- Preserve access for legacy API keys that were already bound to an active
-- exclusive standard group before user_allowed_groups became authoritative.
--
-- Only active, non-deleted keys are considered. Subscription groups are
-- intentionally excluded because their access is governed by subscriptions.
INSERT INTO user_allowed_groups (user_id, group_id, created_at)
SELECT DISTINCT k.user_id, k.group_id, NOW()
FROM api_keys AS k
JOIN groups AS g
  ON g.id = k.group_id
LEFT JOIN user_allowed_groups AS uag
  ON uag.user_id = k.user_id
 AND uag.group_id = k.group_id
WHERE k.deleted_at IS NULL
  AND k.status = 'active'
  AND k.group_id IS NOT NULL
  AND g.deleted_at IS NULL
  AND g.status = 'active'
  AND g.is_exclusive IS TRUE
  AND COALESCE(g.subscription_type, '') <> 'subscription'
  AND uag.user_id IS NULL
ON CONFLICT (user_id, group_id) DO NOTHING;
