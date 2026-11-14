-- Los items con las unidades nuevas vuelven a UNIDAD para poder restaurar la restriccion anterior.
UPDATE inventario_items SET unidad_medida = 'UNIDAD'
WHERE unidad_medida IN ('LT', 'KG', 'FRASCO', 'PAQUETE', 'SOBRE');
ALTER TABLE inventario_items DROP CONSTRAINT IF EXISTS chk_inventario_items_unidad;
ALTER TABLE inventario_items ADD CONSTRAINT chk_inventario_items_unidad
    CHECK (unidad_medida IN ('UNIDAD', 'ML', 'GR', 'CAJA'));
