CREATE TABLE message_reactions (
    message_id  BIGINT NOT NULL REFERENCES messages(id),
    user_id     BIGINT NOT NULL REFERENCES users(id),
    reaction    TEXT NOT NULL CHECK (reaction IN ('like', 'dislike', 'heart')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, user_id)
);
