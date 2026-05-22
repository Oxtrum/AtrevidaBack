CREATE TABLE IF NOT EXISTS clientes (
    id              SERIAL PRIMARY KEY,
    nombre          VARCHAR(100) NOT NULL,
    apellido        VARCHAR(100) NOT NULL,
    numero_telefono VARCHAR(20) NOT NULL,
    CONSTRAINT uq_clientes_nombre_apellido_telefono
        UNIQUE (nombre, apellido, numero_telefono)
);
