create table websites (
    uuid varchar(64),
    url text,
    title text,
    content text,
    update_time timestamp
);

create unique index websites__url on websites(url);
create unique index websites__uuid on websites(uuid);

create table user_websites (
    uuid varchar(64),
    user_uuid varchar(64),
    access_time timestamp,
    group_name text
);

create unique index user_websites__user_and_uuid on user_websites(user_uuid, uuid);
