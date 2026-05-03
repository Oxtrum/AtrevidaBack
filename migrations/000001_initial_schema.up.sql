-- ── Locales ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS locales (
    id      SERIAL PRIMARY KEY,
    nombre  VARCHAR(100) NOT NULL UNIQUE,
    activo  BOOLEAN NOT NULL DEFAULT TRUE
);

-- ── Categorías ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS categorias (
    id      SERIAL PRIMARY KEY,
    nombre  VARCHAR(100) NOT NULL UNIQUE
);

-- ── Tipos de espacio por local ────────────────────────────────────────────────
-- Define qué tipos de espacio (M, B) tiene cada local y cuántos
CREATE TABLE IF NOT EXISTS tipos_espacio_locales (
    id              SERIAL PRIMARY KEY,
    tipo_espacio    VARCHAR(10) NOT NULL,   -- 'M' | 'B'
    cantidad_espacios INT NOT NULL,
    local_id        INT NOT NULL REFERENCES locales(id) ON DELETE CASCADE,
    UNIQUE (tipo_espacio, local_id)
);

-- ── Servicios ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS servicios (
    id           SERIAL PRIMARY KEY,
    nombre       VARCHAR(200) NOT NULL,
    categoria_id INT REFERENCES categorias(id),
    tiempo       VARCHAR(50),
    costo        NUMERIC(10,2),
    sesiones     INT NOT NULL DEFAULT 1,
    activo       BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS servicio_local (
    servicio_id INT NOT NULL REFERENCES servicios(id) ON DELETE CASCADE,
    local_id    INT NOT NULL REFERENCES locales(id)   ON DELETE CASCADE,
    PRIMARY KEY (servicio_id, local_id)
);

-- ── Combos ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS combos (
    id               SERIAL PRIMARY KEY,
    nombre           VARCHAR(200) NOT NULL,
    categoria_id     INT REFERENCES categorias(id),
    costo_total      NUMERIC(10,2),
    sesiones_totales INT NOT NULL DEFAULT 1,
    activo           BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS combo_local (
    combo_id    INT NOT NULL REFERENCES combos(id)  ON DELETE CASCADE,
    local_id    INT NOT NULL REFERENCES locales(id) ON DELETE CASCADE,
    PRIMARY KEY (combo_id, local_id)
);

CREATE TABLE IF NOT EXISTS combo_servicios (
    id          SERIAL PRIMARY KEY,
    combo_id    INT NOT NULL REFERENCES combos(id)    ON DELETE CASCADE,
    servicio_id INT NOT NULL REFERENCES servicios(id) ON DELETE RESTRICT,
    tiempo      VARCHAR(50),
    costo       NUMERIC(10,2),
    sesiones    INT NOT NULL DEFAULT 1,
    orden       INT NOT NULL DEFAULT 0
);

-- ── Planes (por ver, para trazabilidad) ───────────────────────────────
CREATE TABLE IF NOT EXISTS planes (
    id               SERIAL PRIMARY KEY,
    cliente          VARCHAR(200) NOT NULL,
    local_id         INT REFERENCES locales(id),
    combo_id         INT REFERENCES combos(id) ON DELETE SET NULL,
    combo_nombre     VARCHAR(200),             --al contratar
    sesiones_totales INT NOT NULL DEFAULT 1,
    sesiones_usadas  INT NOT NULL DEFAULT 0,
    costo_total      NUMERIC(10,2),
    notas            TEXT,
    activo           BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Reservas (registro histórico desnormalizado) ──────────────────────────────
CREATE TABLE IF NOT EXISTS reservas (
    id               SERIAL PRIMARY KEY,
    local_id         INT REFERENCES locales(id) ON DELETE RESTRICT,
    local_nombre     VARCHAR(100) NOT NULL,   
    tipo_espacio     CHAR(1) NOT NULL,        -- 'M' | 'B'
    fecha            DATE NOT NULL,
    hora_desde       TIME NOT NULL,
    hora_hasta       TIME NOT NULL,
    cliente          VARCHAR(200) NOT NULL,
    plan_id          INT REFERENCES planes(id) ON DELETE SET NULL,
    servicio_nombre  VARCHAR(200),            
    servicio_tiempo  VARCHAR(50),             
    precio           NUMERIC(10,2),           
    notas            TEXT,
    activo           BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tipo_espacio_valido CHECK (tipo_espacio IN ('M', 'B'))
);

CREATE TABLE IF NOT EXISTS detalle_reservas (
    id               SERIAL PRIMARY KEY,
    reserva_id       INT NOT NULL REFERENCES reservas(id) ON DELETE CASCADE,
    servicio_nombre  VARCHAR(200) NOT NULL,
    servicio_tiempo  VARCHAR(50),
    precio           NUMERIC(10,2),
    sesiones         INT NOT NULL DEFAULT 1,
    notas            TEXT
);

CREATE INDEX idx_reservas_fecha        ON reservas(fecha);
CREATE INDEX idx_reservas_local_fecha  ON reservas(local_id, fecha);
CREATE INDEX idx_reservas_cliente      ON reservas(cliente);
CREATE INDEX idx_reservas_plan         ON reservas(plan_id);
CREATE INDEX idx_planes_cliente        ON planes(cliente);
CREATE INDEX idx_planes_combo          ON planes(combo_id);
