INSERT INTO et_resource(name, description)
SELECT 'myProfile', 'Read and edit own profile'
WHERE NOT EXISTS (SELECT 1 FROM et_resource WHERE lower(name) = 'myprofile');

-- Preserve profile access that was previously authorized through myBookings.
INSERT INTO st_group_resource(group_id, resource_id, access_level_id, access_scope_id)
SELECT DISTINCT old_grant.group_id, profile.id, method.id, own_scope.id
FROM st_group_resource old_grant
JOIN et_resource bookings ON bookings.id = old_grant.resource_id AND lower(bookings.name) = 'mybookings'
JOIN et_access_level method ON method.id = old_grant.access_level_id AND upper(method.name) IN ('GET', 'PUT')
JOIN et_access_scope old_scope ON old_scope.id = old_grant.access_scope_id AND lower(old_scope.name) IN ('own', 'any')
JOIN et_access_scope own_scope ON lower(own_scope.name) = 'own'
JOIN et_resource profile ON lower(profile.name) = 'myprofile'
WHERE NOT EXISTS (
    SELECT 1 FROM st_group_resource existing
    WHERE existing.group_id = old_grant.group_id
        AND existing.resource_id = profile.id
        AND existing.access_level_id = method.id
);
