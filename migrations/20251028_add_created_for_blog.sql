alter table tags
    add column created_by uuid references users(id),
    add column updated_by uuid references users(id);

alter table blogs
    add column created_by uuid references users(id),
    add column updated_by uuid references users(id);

-- Triggers to update usage_count in tags table when blog_tags are inserted or deleted
create or replace function update_tag_usage_count()
returns trigger
as $$
BEGIN
  UPDATE tags
  SET usage_count = (
    SELECT COUNT(*)
    FROM blog_tags
    WHERE blog_tags.tag_id = NEW.tag_id
  )
  WHERE id = NEW.tag_id;
  RETURN NULL;
END;
$$
language plpgsql
;

CREATE TRIGGER blog_tags_update_usage
AFTER INSERT OR DELETE ON blog_tags
FOR EACH ROW EXECUTE FUNCTION update_tag_usage_count();

