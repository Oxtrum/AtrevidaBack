CREATE TABLE IF NOT EXISTS usuarios (
    id       SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    password VARCHAR(255) NOT NULL,
    activo   BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_usuarios_username_activo
    ON usuarios(LOWER(username))
    WHERE activo = TRUE;
