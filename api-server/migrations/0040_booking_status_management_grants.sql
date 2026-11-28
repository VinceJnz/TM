-- Booking status definitions are system administration. Booking reads expose
-- their status through the booking resource and do not use this catalog grant.
DELETE FROM st_group_resource grant_row USING st_group group_row, et_resource resource
WHERE grant_row.group_id = group_row.id
  AND grant_row.resource_id = resource.id
  AND group_row.system_key IN ('admin', 'user')
  AND lower(resource.name) = 'bookingstatus';
