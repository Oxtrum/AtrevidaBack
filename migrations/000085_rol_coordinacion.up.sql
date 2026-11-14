-- Rol coordinacion: opera todos los locales y administra catalogo e inventario, sin acceso a datos
-- financieros (finanzas, reportes con dinero, RRHH), ni a usuarios o parametros del sistema.
-- No lleva local asignado, igual que admin_sys.
INSERT INTO roles (codigo, nombre_rol, descripcion)
SELECT 'coordinacion', 'Coordinadora',
       'Opera todos los locales y administra catalogo e inventario. Sin acceso a datos financieros, usuarios ni parametros.'
WHERE NOT EXISTS (
    SELECT 1 FROM roles WHERE LOWER(codigo) = LOWER('coordinacion')
);
