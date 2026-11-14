package models

import "strings"

// Codigos de rol tal como estan en la tabla roles.
const (
	// RolAdminSys puede hacer todo, incluido lo financiero.
	RolAdminSys = "admin_sys"
	// RolCoordinacion opera todos los locales y administra catalogo e inventario, sin acceso a datos
	// financieros (finanzas, reportes con dinero, RRHH), ni a usuarios o parametros del sistema.
	RolCoordinacion = "coordinacion"
	// RolGerencia opera el dia a dia de su local (en pantalla: Recepcion).
	RolGerencia = "gerencia"
)

// EsRolGlobal indica si el rol trabaja sobre todos los locales y por eso no lleva local asignado.
func EsRolGlobal(rolCodigo string) bool {
	rol := strings.TrimSpace(rolCodigo)
	return strings.EqualFold(rol, RolAdminSys) || strings.EqualFold(rol, RolCoordinacion)
}
