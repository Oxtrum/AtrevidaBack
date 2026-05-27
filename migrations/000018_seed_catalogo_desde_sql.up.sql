-- Datos reemplazo del antiguo flujo /admin/importar basado en Google Sheets.
-- Se limpian solo tablas de catalogo para conservar datos operativos como reservas/clientes.

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

INSERT INTO categorias (nombre) VALUES
	 ('SERVICIOS'),
	 ('SERVICIOS MANUALES'),
	 ('COMBOS INYECCIONES + APARATOS'),
	 ('COMBOS'),
	 ('LIMPIEZA FACIAL'),
	 ('INYECCIONES');

INSERT INTO servicios (nombre,categoria_id,tiempo,costo,sesiones,activo,tipo_espacio_requerido,requiere_evaluacion) VALUES
	 ('E - Pulse Bike',1,'30 min',70.00,1,true,'B',false),
	 ('Vacumterapia premium',1,'50 min',70.00,1,true,'M',true),
	 ('Criolipolisis',1,'50 min',500.00,1,true,'M',true),
	 ('Vacumterapia',1,'50 min',70.00,1,true,'M',true),
	 ('Lipolaser',1,'50 min',70.00,1,true,'M',true),
	 ('Ondas Rusas',1,'50 min',70.00,1,true,'M',true),
	 ('Radiofrencia',1,'50 min',70.00,1,true,'M',true),
	 ('Ultracavitación',1,'50 min',70.00,1,true,'M',true),
	 ('Masaje reductor',2,'25 Min',50.00,1,true,'M',true),
	 ('Masaje descontracturante cuerpo entero',2,'50 min',100.00,1,true,'M',true),
	 ('Masaje descontracturante medio cuerpo',2,'35 min',70.00,1,true,'M',true),
	 ('Masaje relajante cuerpo entero',2,'50 min',100.00,1,true,'M',true),
	 ('Masaje relajante medio cuerpo',2,'35 min',70.00,1,true,'M',true),
	 ('Maderoterapia',2,'50 min',70.00,1,true,'M',true),
	 ('Limpieza Facial Premium',5,'50Min',150.00,1,true,'M',false),
	 ('Limpieza Facial Premium',5,'1 hora y 30 min',150.00,1,true,'M',false),
	 ('Limpieza Facial',5,'50 min',100.00,1,true,'M',false),
	 ('DMAE reafirmante tensor + radiofrecuencia + masaje tonificador',6,'50 min',2900.00,1,true,'M',true),
	 ('DMAE reafirmante tensor + radiofrecuencia + masaje tonificador',6,'50 min',1500.00,1,true,'M',true),
	 ('DMAE reafirmante tensor + radiofrecuencia + masaje tonificador',6,'50 min',900.00,1,true,'M',true),
	 ('DMAE reafirmante tensor + radiofrecuencia + masaje tonificador',6,'50 min',350.00,1,true,'M',true),
	 ('Quemador de grasa + ultracavitación + masaje reductor',6,'50 min',2350.00,1,true,'M',true),
	 ('Quemador de grasa + ultracavitación + masaje reductor',6,'50 min',1250.00,1,true,'M',true),
	 ('Quemador de grasa + ultracavitación + masaje reductor',6,'50 min',750.00,1,true,'M',true),
	 ('Quemador de grasa + ultracavitación + masaje reductor',6,'50 min',300.00,1,true,'M',true);

INSERT INTO servicio_local (servicio_id,local_id) VALUES
	 (1,1),
	 (2,2),
	 (3,2),
	 (4,1),
	 (4,2),
	 (5,1),
	 (5,2),
	 (6,1),
	 (6,2),
	 (7,1),
	 (7,2),
	 (8,1),
	 (8,2),
	 (9,1),
	 (10,1),
	 (10,2),
	 (11,1),
	 (11,2),
	 (12,1),
	 (12,2),
	 (13,1),
	 (13,2),
	 (14,1),
	 (14,2),
	 (15,1),
	 (16,2),
	 (17,1),
	 (17,2),
	 (18,1),
	 (18,2),
	 (19,1),
	 (19,2),
	 (20,1),
	 (20,2),
	 (21,1),
	 (21,2),
	 (22,1),
	 (22,2),
	 (23,1),
	 (23,2),
	 (24,1),
	 (24,2),
	 (25,1),
	 (25,2);

INSERT INTO combos (nombre,categoria_id,costo_total,sesiones_totales,activo,descripcion,creado_en,actualizado_en) VALUES
	 ('peptonas + ondas rusas',3,200.00,1,true,NULL,'2026-05-26 20:28:47.967319-04','2026-05-26 20:28:47.967319-04'),
	 ('PEPTONAS',3,250.00,1,true,NULL,'2026-05-26 20:28:47.967319-04','2026-05-26 20:28:47.967319-04'),
	 ('PEPTONAS PREMIUM',3,250.00,1,true,NULL,'2026-05-26 20:28:47.967319-04','2026-05-26 20:28:47.967319-04'),
	 ('PROMO CRIOLIPOLISIS',3,750.00,1,true,NULL,'2026-05-26 20:28:47.967319-04','2026-05-26 20:28:47.967319-04'),
	 ('PAQUETE ABDOMEN PREMIUM',3,700.00,1,true,NULL,'2026-05-26 20:28:47.967319-04','2026-05-26 20:28:47.967319-04');

INSERT INTO combo_local (combo_id,local_id) VALUES
	 (1,1),
	 (4,2),
	 (2,2),
	 (5,2),
	 (5,1),
	 (3,2);

INSERT INTO combo_servicios (combo_id,servicio_id,tiempo,costo,sesiones,orden,servicio_texto,activo) VALUES
	 (1,NULL,'50 min',200.00,1,0,'peptonas + ondas rusas',true),
	 (2,NULL,'50 min',200.00,1,0,'peptonas + ondas rusas',true),
	 (3,NULL,'50 min',NULL,1,0,'Vacumterapia premium + peptonas + ondas rusas',true),
	 (4,NULL,'50 min',NULL,1,1,'DMAE reafirmante tensor + radiofrecuencia + masaje tonificador',true),
	 (4,NULL,'50 min',NULL,1,0,'Criolipolisis',true),
	 (5,NULL,'50 min',NULL,1,4,'Radiofrencia + Ondas Rusas',true),
	 (5,NULL,'50 min',NULL,1,3,'Ultracavitación + Vacumterapia',true),
	 (5,NULL,'50 min',NULL,1,2,'Lipolaser + Ultracavitación',true),
	 (5,NULL,'50 min',NULL,1,1,'Quemador de grasa + ultracavitación + masaje reductor',true),
	 (5,NULL,'50 min',NULL,1,0,'Quemador de grasa + ultracavitación + masaje reductor',true);
