UPDATE reservas
SET numero_telefono = '77777777'
WHERE numero_telefono IS NULL
   OR BTRIM(numero_telefono) = '';
