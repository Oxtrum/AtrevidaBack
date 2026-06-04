ALTER TABLE usuarios
    DROP CONSTRAINT IF EXISTS fk_usuarios_roles;

ALTER TABLE usuarios
    DROP COLUMN IF EXISTS rol_id;
