-- Acelera el detalle, el historial y los filtros de consumos por insumo.
CREATE INDEX IF NOT EXISTS idx_atencion_insumos_atencion
    ON atencion_insumos (atencion_id);

CREATE INDEX IF NOT EXISTS idx_atencion_insumos_item
    ON atencion_insumos (item_id, atencion_id)
    WHERE item_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_atencion_insumos_pendientes
    ON atencion_insumos (atencion_id)
    WHERE estado_descuento = 'PENDIENTE_STOCK';
