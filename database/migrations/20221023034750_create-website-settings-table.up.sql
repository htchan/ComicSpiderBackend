create table website_settings (
  domain varchar(255),
  title_regex text,
  content_regex text,
  focus_index_from int,
  focus_index_to int
);

create unique index website_settings__domain on website_settings(domain);

insert into website_settings 
(domain, title_regex, content_regex, focus_index_from, focus_index_to) values
('default', '<title.*?>(?P<Title>.*?)</title>', '(?P<Content>(\d{2,4}[-/年])?(1[0-2]|0?[1-9])[-/月](1[0-9]|2[0-9]|3[0-1]|0?[1-9])[日號号]?)', 0, 2)