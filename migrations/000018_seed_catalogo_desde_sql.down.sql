DELETE FROM combo_servicios;
DELETE FROM combo_local;
DELETE FROM servicio_local;
DELETE FROM combos;
DELETE FROM servicios;
DELETE FROM categorias;

ALTER SEQUENCE IF EXISTS categorias_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS servicios_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS combos_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS combo_servicios_id_seq RESTART WITH 1;
