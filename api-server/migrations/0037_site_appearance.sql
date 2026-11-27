CREATE TABLE st_appearance_revision (
    version bigint PRIMARY KEY CHECK (version > 0),
    settings jsonb NOT NULL,
    published_by integer REFERENCES st_users(id),
    published_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO st_appearance_revision(version, settings) VALUES (1, '{}');

INSERT INTO et_resource(name)
SELECT 'siteAppearance' WHERE NOT EXISTS (SELECT 1 FROM et_resource WHERE lower(name)='siteappearance');
INSERT INTO st_group_resource(group_id, resource_id, access_level_id, access_scope_id)
SELECT DISTINCT existing.group_id, appearance.id, method.id, scope.id
FROM st_group_resource existing
JOIN et_resource administrative ON administrative.id=existing.resource_id
JOIN et_access_scope scope ON scope.id=existing.access_scope_id
JOIN et_access_level current_method ON current_method.id=existing.access_level_id
CROSS JOIN et_resource appearance
CROSS JOIN et_access_level method
WHERE lower(administrative.name)='securitygroupresource' AND lower(current_method.name)='put'
  AND lower(scope.name)='any' AND lower(appearance.name)='siteappearance' AND lower(method.name) IN ('get','put')
  AND NOT EXISTS (
    SELECT 1 FROM st_group_resource present WHERE present.group_id=existing.group_id
      AND present.resource_id=appearance.id AND present.access_level_id=method.id
      AND present.access_scope_id=scope.id
  );
