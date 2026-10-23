-- Bonos parametrizables: catalogo de tipos y detalle por planilla (planillas.bonos queda como la suma de las lineas).
CREATE TABLE tipos_bono (
    id                 SERIAL PRIMARY KEY,
    nombre             VARCHAR(100) NOT NULL,
    descripcion        TEXT,
    activo             BOOLEAN NOT NULL DEFAULT TRUE,
    fecha_creacion     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_tipos_bono_nombre ON tipos_bono (LOWER(nombre));

CREATE TABLE planilla_bonos (
    id           SERIAL PRIMARY KEY,
    planilla_id  INT NOT NULL REFERENCES planillas(id) ON DELETE CASCADE,
    tipo_bono_id INT NOT NULL REFERENCES tipos_bono(id) ON DELETE RESTRICT,
    monto        NUMERIC(12,2) NOT NULL,
    nota         TEXT,
    CONSTRAINT chk_planilla_bonos_monto CHECK (monto > 0)
);
CREATE INDEX idx_planilla_bonos_planilla ON planilla_bonos (planilla_id);

INSERT INTO tipos_bono (nombre, descripcion) VALUES
    ('Producción', 'Bono por metas o servicios realizados en el mes.'),
    ('Puntualidad', 'Bono por asistencia y puntualidad.'),
    ('Otros bonos', 'Bonos sin clasificar.');

-- Los bonos ya cargados como un solo numero pasan a una linea "Otros bonos".
INSERT INTO planilla_bonos (planilla_id, tipo_bono_id, monto, nota)
SELECT p.id, (SELECT id FROM tipos_bono WHERE nombre = 'Otros bonos'), p.bonos, 'Registrado antes del catálogo de bonos'
FROM planillas p
WHERE p.bonos > 0;
