package app

import (
	"errors"
	"fmt"
	"std/internal/model"
	"strings"

	"gorm.io/gorm"
)

// nextProcedureNumber reserva el siguiente correlativo y arma el numero visible del tramite:
//   - PERSONAL:   PERSONAL/{correlativo 2 digitos}/{ID usuario 4 digitos} -> PERSONAL/01/0101
//   - DEPARTMENT: TRAMITE/{codigo del area}/{correlativo}                  -> TRAMITE/SYS/1
//
// Debe llamarse dentro de la transaccion que crea el tramite: si esta falla, la reserva se revierte.
// El upsert con incremento es atomico, asi que no hay duplicados entre peticiones concurrentes.
func nextProcedureNumber(tx *gorm.DB, p model.Procedure) (string, error) {
	var scope string
	switch {
	case p.ProcedureType == "DEPARTMENT" && p.DepartmentID != nil:
		scope = fmt.Sprintf("DEPARTMENT:%d", *p.DepartmentID)
	case p.ProcedureType == "PERSONAL":
		scope = fmt.Sprintf("USER:%d", p.CreatedBy)
	default:
		return "", errors.New("invalid procedure type")
	}
	var n int
	if e := tx.Raw(`INSERT INTO procedure_sequences(scope_key,last_number) VALUES (?,1)
		ON CONFLICT (scope_key) DO UPDATE SET last_number=procedure_sequences.last_number+1, updated_at=now()
		RETURNING last_number`, scope).Scan(&n).Error; e != nil {
		return "", e
	}
	if p.ProcedureType == "PERSONAL" {
		return fmt.Sprintf("PERSONAL/%02d/%04d", n, p.CreatedBy), nil
	}
	var code string
	if e := tx.Model(&model.Department{}).Select("code").Where("id=?", *p.DepartmentID).Scan(&code).Error; e != nil {
		return "", e
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return "", errors.New("department not found or has no code")
	}
	return fmt.Sprintf("TRAMITE/%s/%d", code, n), nil
}
