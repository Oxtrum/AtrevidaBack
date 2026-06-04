ALTER TABLE usuarios
    ADD COLUMN IF NOT EXISTS rol_id INT;

UPDATE usuarios
SET rol_id = (
    SELECT id
    FROM roles
    WHERE LOWER(codigo) = LOWER('admin_sys')
)
WHERE rol_id IS NULL;

ALTER TABLE usuarios
    ALTER COLUMN rol_id SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_usuarios_roles'
          AND conrelid = 'usuarios'::regclass
    ) THEN
        ALTER TABLE usuarios
            ADD CONSTRAINT fk_usuarios_roles
            FOREIGN KEY (rol_id)
            REFERENCES roles(id);
    END IF;
END $$;
