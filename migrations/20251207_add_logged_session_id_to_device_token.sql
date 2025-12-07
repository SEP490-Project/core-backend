ALTER TABLE device_tokens
ADD COLUMN logged_session_id UUID REFERENCES logged_sessions(id) ON DELETE CASCADE;

CREATE INDEX idx_device_tokens_logged_session_id ON device_tokens(logged_session_id);

create index idx_logged_sessions_refresh_token_hash on logged_sessions(refresh_token_hash);

