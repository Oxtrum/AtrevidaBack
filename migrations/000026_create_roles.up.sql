CREATE TABLE IF NOT EXISTS roles (
    id          SERIAL PRIMARY KEY,
    codigo      VARCHAR(50) NOT NULL,
    nombre_rol  VARCHAR(100) NOT NULL,
    descripcion TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_codigo
    ON roles(LOWER(codigo));

INSERT INTO roles (codigo, nombre_rol, descripcion)
SELECT 'admin_sys', 'Administrador de sistema', 'Puede hacer todo dentro del sistema.'
WHERE NOT EXISTS (
    SELECT 1 FROM roles WHERE LOWER(codigo) = LOWER('admin_sys')
);

INSERT INTO roles (codigo, nombre_rol, descripcion)
SELECT 'gerencia', 'Gerencia', 'Puede hacer todo excepto acceder a los reportes de pagos.'
WHERE NOT EXISTS (
    SELECT 1 FROM roles WHERE LOWER(codigo) = LOWER('gerencia')
);
