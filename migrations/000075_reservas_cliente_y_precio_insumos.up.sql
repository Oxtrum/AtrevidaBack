-- Reservas con cliente explicito (PLAN_CUENTA_CLIENTE_Y_AJUSTES D30) e insumos con precio de venta (D31).

-- Una reserva sin cliente_id esta "por vincular". Si el cliente se borra, la reserva vuelve a esa lista.
ALTER TABLE reservas
    ADD COLUMN IF NOT EXISTS cliente_id INT REFERENCES clientes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_reservas_cliente_id ON reservas (cliente_id) WHERE cliente_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_reservas_por_vincular ON reservas (fecha DESC, id DESC) WHERE cliente_id IS NULL;

COMMENT ON COLUMN reservas.cliente_id IS
    'Cliente registrado dueño de la reserva. NULL = por vincular; nada se vincula por adivinanza.';

-- Historico: solo se enlazan las reservas cuyo telefono E.164 coincide exacto con UN unico cliente.
-- Si dos clientes comparten el telefono no se elige ninguno: la reserva queda por vincular.
UPDATE reservas r
SET cliente_id = c.id
FROM (
    SELECT telefono_e164, MIN(id) AS id
    FROM clientes
    WHERE telefono_e164 IS NOT NULL
    GROUP BY telefono_e164
    HAVING COUNT(*) = 1
) c
WHERE r.cliente_id IS NULL
  AND r.telefono_e164 IS NOT NULL
  AND r.telefono_e164 = c.telefono_e164;

-- Precio al que se vende el insumo; costo_referencia sigue siendo lo que cuesta. NULL = sin precio definido.
ALTER TABLE inventario_items
    ADD COLUMN IF NOT EXISTS precio_venta NUMERIC(12,2);

ALTER TABLE inventario_items
    ADD CONSTRAINT chk_inventario_items_precio_venta
        CHECK (precio_venta IS NULL OR precio_venta >= 0);

COMMENT ON COLUMN inventario_items.precio_venta IS
    'Precio de venta sugerido del insumo al registrarlo como consumo o desde la caja; editable al vender.';
