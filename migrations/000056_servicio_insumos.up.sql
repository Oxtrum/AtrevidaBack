CREATE TABLE servicio_insumos (
    servicio_id         INT NOT NULL REFERENCES servicios(id) ON DELETE CASCADE,
    item_id             INT NOT NULL REFERENCES inventario_items(id) ON DELETE RESTRICT,
    cantidad_por_sesion NUMERIC(12,3) NOT NULL,
    PRIMARY KEY (servicio_id, item_id),
    CONSTRAINT chk_servicio_insumos_cantidad CHECK (cantidad_por_sesion > 0)
);
CREATE INDEX idx_servicio_insumos_item ON servicio_insumos (item_id);

COMMENT ON TABLE servicio_insumos IS 'Insumos de inventario que consume un servicio por sesion (RN-10).';
