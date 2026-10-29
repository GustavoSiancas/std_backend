package app

import (
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

// reserveDocumentNumber must run in the same transaction as document creation.
// The advisory lock also serializes the first insert for a scope not yet created.
func reserveDocumentNumber(tx *gorm.DB, typeID int64, departmentID *int64, year int, code string) (string, error) {
	seq, err := reserveSequence(tx, typeID, departmentID, year)
	if err != nil {
		return "", err
	}
	return formatDocumentNumber(seq.LastNumber, code), nil
}

func formatDocumentNumber(correlative int, departmentCode string) string {
	return fmt.Sprintf("STD-%d-%s", correlative, strings.ToUpper(strings.TrimSpace(departmentCode)))
}

type reservedSequence struct {
	ID         int64
	LastNumber int
}

func reserveSequence(tx *gorm.DB, typeID int64, departmentID *int64, year int) (reservedSequence, error) {
	scope := "GLOBAL"
	if departmentID != nil {
		scope = strconv.FormatInt(*departmentID, 10)
	}
	key := fmt.Sprintf("%d/%s/%d", typeID, scope, year)
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error; err != nil {
		return reservedSequence{}, err
	}
	var seq reservedSequence
	err := tx.Raw(`INSERT INTO document_sequences(document_type_id,department_id,year,last_number)
 VALUES (?,?,?,1)
 ON CONFLICT (document_type_id,(COALESCE(department_id,0)),year)
 DO UPDATE SET last_number=document_sequences.last_number+1,updated_at=now()
 RETURNING id,last_number`, typeID, departmentID, year).Scan(&seq).Error
	if err != nil {
		return reservedSequence{}, err
	}
	return seq, nil
}
