-- Mas unidades de medida para inventario: litros, kilogramos, frascos, paquetes y sobres.
ALTER TABLE inventario_items DROP CONSTRAINT IF EXISTS chk_inventario_items_unidad;
ALTER TABLE inventario_items ADD CONSTRAINT chk_inventario_items_unidad
    CHECK (unidad_medida IN ('UNIDAD', 'ML', 'GR', 'LT', 'KG', 'FRASCO', 'PAQUETE', 'SOBRE', 'CAJA'));
