create table user_websites (
    website_uuid varchar(64),
    user_uuid varchar(64),
    access_time timestamp,
    group_name text
);

create unique index user_websites__user_and_uuid on user_websites(user_uuid, website_uuid);
