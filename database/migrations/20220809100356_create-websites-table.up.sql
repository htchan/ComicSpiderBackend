create table websites (
    uuid varchar(64),
    url text,
    title text,
    content text,
    update_time timestamp
);

create unique index websites__url on websites(url);
create unique index websites__uuid on websites(uuid);
