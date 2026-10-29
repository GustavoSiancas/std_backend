package app

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"std/internal/model"
	"strings"
	"time"
	_ "time/tzdata"
)

type CreateDocumentInput struct {
	DocumentTypeID int64   `json:"document_type_id"`
	DepartmentID   int64   `json:"department_id"`
	CreatedBy      int64   `json:"created_by"`
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	FileURL        string  `json:"file_url"`
}

var errDocumentInput = errors.New("name, valid file_url, active document_type_id, department_id and created_by are required")
var errDocumentMembership = errors.New("user must have active membership in the document department")

// CreateDocument reserves and snapshots the counter atomically with the document.
// created_by is client supplied in the current token-free workflow.
func (a *App) CreateDocument(ctx context.Context, in CreateDocumentInput) (model.Document, error) {
	in.Name = strings.TrimSpace(in.Name)
	link, err := validateDocumentURL(in.FileURL)
	if err != nil || in.Name == "" || len(in.Name) > 255 || in.DocumentTypeID == 0 || in.DepartmentID == 0 || in.CreatedBy == 0 {
		return model.Document{}, errDocumentInput
	}
	var doc model.Document
	err = a.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dt model.DocumentType
		if err := tx.Where("id=? AND is_active", in.DocumentTypeID).First(&dt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDocumentInput
			}
			return err
		}
		for _, entry := range []struct {
			model any
			id    int64
		}{{&model.User{}, in.CreatedBy}} {
			var n int64
			if err := tx.Model(entry.model).Where("id=? AND is_active", entry.id).Count(&n).Error; err != nil {
				return err
			}
			if n != 1 {
				return errDocumentInput
			}
		}
		var department model.Department
		if err := tx.Where("id=? AND is_active", in.DepartmentID).First(&department).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDocumentInput
			}
			return err
		}
		var memberships int64
		if err := tx.Model(&model.DepartmentMember{}).Where("user_id=? AND department_id=? AND is_active", in.CreatedBy, in.DepartmentID).Count(&memberships).Error; err != nil {
			return err
		}
		if memberships == 0 {
			return errDocumentMembership
		}
		doc = model.Document{DocumentTypeID: in.DocumentTypeID, DepartmentID: &in.DepartmentID, CreatedBy: in.CreatedBy, Name: in.Name, Description: in.Description, FileURL: &link}
		if dt.HasCorrelative {
			zone, err := time.LoadLocation("America/Bogota")
			if err != nil {
				return err
			}
			seq, err := reserveSequence(tx, in.DocumentTypeID, &in.DepartmentID, time.Now().In(zone).Year())
			if err != nil {
				return err
			}
			doc.DocumentSequenceID = &seq.ID
			doc.CorrelativeNumber = &seq.LastNumber
			formatted := formatDocumentNumber(seq.LastNumber, department.Code)
			doc.DocumentNumber = &formatted
		}
		if err := tx.Create(&doc).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{ActorID: in.CreatedBy, EntityType: "DOCUMENT", EntityID: doc.ID, Action: "DOCUMENT_CREATED", Metadata: "{}"}).Error
	})
	return doc, err
}
