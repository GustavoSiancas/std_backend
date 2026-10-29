package migrations

import (
	"fmt"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
)

func TestSchemaIntegrity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integrity tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	schema := "std_test_" + uuid.NewString()[:8]
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema + ",public").Error; err != nil {
		t.Fatal(err)
	}
	if err := Apply(tx); err != nil {
		t.Fatal(err)
	}
	if err := Apply(tx); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	var generatedID int64
	if err := tx.Raw("INSERT INTO document_types(name,code,has_correlative) VALUES ('Generated','GENERATED_ID',false) RETURNING id").Scan(&generatedID).Error; err != nil || generatedID < 1 {
		t.Fatalf("numeric identity was not generated: id=%d err=%v", generatedID, err)
	}
	user := int64(1001)
	if err := tx.Exec("INSERT INTO users(id,name,email) VALUES (?, 'Test','test@example.test')", user).Error; err != nil {
		t.Fatal(err)
	}
	dep := int64(1002)
	if err := tx.Exec("INSERT INTO departments(id,name,code) VALUES (?, 'Test','TEST')", dep).Error; err != nil {
		t.Fatal(err)
	}
	msg := int64(1003)
	if err := tx.Exec("INSERT INTO messages(id,created_by,sender_user_id,message_type,visibility,subject,body) VALUES (?,?,?,'PERSONAL','PRIVATE','Test','Body')", msg, user, user).Error; err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"foreign key":         "INSERT INTO messages(created_by,sender_user_id,message_type,visibility,subject,body) VALUES (99999999,99999999,'PERSONAL','PRIVATE','X','X')",
		"department required": fmt.Sprintf("INSERT INTO messages(created_by,message_type,visibility,subject,body) VALUES (%d,'DEPARTMENT','PRIVATE','X','X')", user),
		"personal visibility": fmt.Sprintf("INSERT INTO messages(created_by,sender_user_id,message_type,visibility,subject,body) VALUES (%d,%d,'PERSONAL','DEPARTMENT','X','X')", user, user),
		"recipient missing":   fmt.Sprintf("INSERT INTO message_recipients(message_id,type) VALUES (%d,'TO')", msg),
		"recipient both":      fmt.Sprintf("INSERT INTO message_recipients(message_id,recipient_id,department_id,type) VALUES (%d,%d,%d,'TO')", msg, user, dep),
		"self relation":       fmt.Sprintf("INSERT INTO message_relations(source_message_id,target_message_id,relation_type,created_by) VALUES (%d,%d,'REPLY',%d)", msg, msg, user),
	}
	for name, sql := range cases {
		t.Run(name, func(t *testing.T) {
			tx.SavePoint("invalid_case")
			if err := tx.Exec(sql).Error; err == nil {
				t.Error("expected database constraint violation")
			}
			tx.RollbackTo("invalid_case")
		})
	}
	dt := int64(1004)
	if err := tx.Exec("INSERT INTO document_types(id,name,code,has_correlative) VALUES (?,'Test','TEST_SEQUENCE',true)", dt).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("INSERT INTO document_sequences(document_type_id,year) VALUES (?,2026)", dt).Error; err != nil {
		t.Fatal(err)
	}
	tx.SavePoint("duplicate_seq")
	if err := tx.Exec("INSERT INTO document_sequences(document_type_id,year) VALUES (?,2026)", dt).Error; err == nil {
		t.Error("duplicate global sequence accepted")
	}
	tx.RollbackTo("duplicate_seq")
}
