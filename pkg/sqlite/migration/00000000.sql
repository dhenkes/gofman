CREATE TABLE IF NOT EXISTS users (
  id          UUID PRIMARY KEY,
  username    VARCHAR(35) NOT NULL UNIQUE,
  password    VARCHAR(255) NOT NULL,
  is_admin    BOOLEAN NOT NULL DEFAULT FALSE,
  is_demo     BOOLEAN NOT NULL DEFAULT FALSE,
  created_at  BIGINT NOT NULL,
  updated_at  BIGINT NOT NULL,
  removed_at  BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sessions (
  id          UUID PRIMARY KEY,
  users_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  token       VARCHAR(255) NOT NULL,
  created_at  BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
  id          UUID PRIMARY KEY,
  users_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  name        TEXT NOT NULL,
  type        VARCHAR(255) NOT NULL,
  path        TEXT NOT NULL,
  checksum    TEXT NOT NULL,
  created_at  BIGINT NOT NULL,
  updated_at  BIGINT NOT NULL,
  removed_at  BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tags (
  id          UUID PRIMARY KEY,
  users_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  name        VARCHAR(255) NOT NULL,
  created_at  BIGINT NOT NULL,
  updated_at  BIGINT NOT NULL,
  removed_at  BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS actors (
  id          UUID PRIMARY KEY,
  users_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  name        VARCHAR(255) NOT NULL,
  created_at  BIGINT NOT NULL,
  updated_at  BIGINT NOT NULL,
  removed_at  BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS files_actors (
  files_id   UUID NOT NULL REFERENCES files(id) ON DELETE RESTRICT,
  actors_id  UUID NOT NULL REFERENCES actors(id) ON DELETE RESTRICT,
  PRIMARY KEY (files_id, actors_id)
);

CREATE TABLE IF NOT EXISTS files_tags (
  files_id  UUID NOT NULL REFERENCES files(id) ON DELETE RESTRICT,
  tags_id   UUID NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
  PRIMARY KEY (files_id, tags_id)
);

CREATE TABLE IF NOT EXISTS actors_tags (
  actors_id  UUID NOT NULL REFERENCES actors(id) ON DELETE RESTRICT,
  tags_id    UUID NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
  PRIMARY KEY (actors_id, tags_id)
);
