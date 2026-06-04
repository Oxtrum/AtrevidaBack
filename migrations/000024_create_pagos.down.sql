DROP INDEX IF EXISTS idx_detalle_pagos_pago;
DROP INDEX IF EXISTS idx_pagos_estado;
DROP INDEX IF EXISTS idx_pagos_cliente;
DROP INDEX IF EXISTS idx_pagos_local;

DROP TABLE IF EXISTS detalle_pagos;
DROP TABLE IF EXISTS pagos;
DROP SEQUENCE IF EXISTS pagos_codigo_pago_seq;
