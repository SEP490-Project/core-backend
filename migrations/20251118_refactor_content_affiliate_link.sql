DO
$$
BEGIN
    alter table affiliate_links
        add column if not exists affiliate_url string NOT NULL DEFAULT '';

    alter table contents
        drop column if exists affiliate_link text;

    atler table content_channels
        add column if not exists affiliate_link_id uuid REFERENCES affiliate_links(id) ON DELETE SET NULL;
END
;
$$
;

