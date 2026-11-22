-- Standard members may browse the global roster, with hidden profiles redacted
-- by the report store. This grants GET only and does not change booking ownership.
INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id)
SELECT g.id,r.id,l.id,s.id
FROM st_group g CROSS JOIN et_resource r CROSS JOIN et_access_level l CROSS JOIN et_access_scope s
WHERE lower(btrim(g.name))='user' AND lower(r.name)='tripsreport'
 AND lower(l.name)='get' AND lower(s.name)='any'
ON CONFLICT DO NOTHING;

-- Retire the unsupported own-report grant only for the standard User group.
DELETE FROM st_group_resource gr
USING st_group g,et_resource r,et_access_level l,et_access_scope s
WHERE gr.group_id=g.id AND gr.resource_id=r.id AND gr.access_level_id=l.id AND gr.access_scope_id=s.id
 AND lower(btrim(g.name))='user' AND lower(r.name)='tripsreport'
 AND lower(l.name)='get' AND lower(s.name)='own';