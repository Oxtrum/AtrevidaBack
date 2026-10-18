ALTER TABLE reservas ADD COLUMN alerta_inventario BOOLEAN NOT NULL DEFAULT FALSE;

-- Alertas del panel: solo reservas activas con aviso de stock.
CREATE INDEX idx_reservas_alerta_inventario ON reservas (fecha, hora_desde)
    WHERE alerta_inventario = TRUE AND activo = TRUE;

COMMENT ON COLUMN reservas.alerta_inventario IS 'true si al crear/agendar faltaba stock de insumos para el servicio (RN-12).';
