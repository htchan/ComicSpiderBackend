-- name: CreateWebsite :one
INSERT INTO websites
(uuid, url, title, content, update_time)
VALUES
($1, $2, $3, $4, $5)
ON CONFLICT (url) DO
UPDATE SET url=$2
RETURNING *;

-- name: UpdateWebsite :one
UPDATE websites SET
url=$1, title=$2, content=$3, update_time=$4
WHERE uuid=$5
RETURNING *;

-- name: DeleteWebsite :exec
DELETE FROM websites WHERE uuid=$1;

-- name: ListWebsites :many
SELECT * FROM websites;

-- name: GetWebsite :one
SELECT * from websites WHERE uuid=$1;

-- name: CreateUserWebsite :one
INSERT INTO user_websites
(user_uuid, website_uuid, access_time, group_name)
VALUES
($1, $2, $3, $4)
ON CONFLICT(user_uuid, website_uuid) DO
UPDATE SET user_uuid=$1, website_uuid=$2
RETURNing *;

-- name: UpdateUserWebsite :one
UPDATE user_websites SET
access_time=$1, group_name=$2
WHERE user_uuid=$3 and website_uuid=$4
RETURNING *;

-- name: DeleteUserWebsite :exec
DELETE FROM user_websites
where user_uuid=$1 and website_uuid=$2;

-- name: ListUserWebsites :many
SELECT website_uuid, user_uuid, access_time, group_name,
uuid, url, title, update_time 
FROM user_websites JOIN websites ON user_websites.website_uuid=websites.uuid 
WHERE user_uuid=$1
ORDER BY (update_time > access_time) DESC, update_time DESC, access_time DESC;

-- name: ListUserWebsitesByGroup :many
SELECT website_uuid, user_uuid, access_time, group_name ,
uuid, url, title, update_time 
FROM user_websites JOIN websites ON user_websites.website_uuid=websites.uuid 
WHERE user_uuid=$1 and group_name=$2;

-- name: GetUserWebsite :one
SELECT website_uuid, user_uuid, access_time, group_name ,
uuid, url, title, update_time 
FROM user_websites JOIN websites ON user_websites.website_uuid=websites.uuid 
WHERE user_uuid=$1 and website_uuid=$2;
