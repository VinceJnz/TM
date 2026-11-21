ALTER TABLE at_provider_refunds ADD COLUMN IF NOT EXISTS first_seen timestamptz;
UPDATE at_provider_refunds SET first_seen=checked_at WHERE first_seen IS NULL;
ALTER TABLE at_provider_refunds ALTER COLUMN first_seen SET NOT NULL,ALTER COLUMN first_seen SET DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE at_financial_audit ADD COLUMN IF NOT EXISTS request_id text;
ALTER TABLE at_financial_audit ALTER COLUMN request_id SET DEFAULT NULLIF(current_setting('tm.request_id',true),'');
INSERT INTO et_resource(name) SELECT 'operations' WHERE NOT EXISTS(SELECT 1 FROM et_resource WHERE name='operations');
-- Grant aggregate operations visibility through existing security-administration
-- capabilities, without relying on a role name or group ID.
INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id)
SELECT DISTINCT existing.group_id,operations.id,method.id,scope.id
FROM st_group_resource existing JOIN et_resource administrative ON administrative.id=existing.resource_id
JOIN et_access_level method ON method.id=existing.access_level_id
JOIN et_access_scope scope ON scope.id=existing.access_scope_id
CROSS JOIN et_resource operations
WHERE administrative.name='securityGroupResource' AND method.name='GET' AND scope.name='any' AND operations.name='operations'
AND NOT EXISTS(SELECT 1 FROM st_group_resource present WHERE present.group_id=existing.group_id AND present.resource_id=operations.id AND present.access_level_id=method.id AND present.access_scope_id=scope.id);
