package app

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"std/internal/model"
	"strconv"
	"strings"
)

func documentSort(c *gin.Context, allowed map[string]string, fallback, direction string) (string, bool) {
	field := c.DefaultQuery("sort_by", fallback)
	column, ok := allowed[field]
	if !ok {
		fail(c, 400, errors.New("invalid sort_by"))
		return "", false
	}
	order := strings.ToLower(c.DefaultQuery("order", direction))
	if order != "asc" && order != "desc" {
		fail(c, 400, errors.New("order must be asc or desc"))
		return "", false
	}
	return column + " " + order + ", " + allowed["id"] + " " + order, true
}
func (a *App) listDocumentTypes(c *gin.Context) {
	p, ok := directoryQuery(c)
	if !ok {
		return
	}
	order, ok := documentSort(c, map[string]string{"id": "id", "name": "name", "code": "code", "created_at": "created_at"}, "name", "asc")
	if !ok {
		return
	}
	q := a.DB.WithContext(c.Request.Context()).Model(&model.DocumentType{})
	if s := strings.TrimSpace(c.Query("q")); s != "" {
		q = q.Where("(name ILIKE ? OR code ILIKE ?)", "%"+s+"%", "%"+s+"%")
	}
	q, ok = boolFilter(c, q, "is_active", "is_active")
	if !ok {
		return
	}
	q, ok = boolFilter(c, q, "has_correlative", "has_correlative")
	if !ok {
		return
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		accountError(c, err)
		return
	}
	rows := make([]model.DocumentType, 0)
	if err := q.Order(order).Offset((p.Page - 1) * p.Size).Limit(p.Size).Find(&rows).Error; err != nil {
		accountError(c, err)
		return
	}
	directoryResult(c, rows, total, p)
}

type documentTypeInfo struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	HasCorrelative bool   `json:"has_correlative"`
}
type documentView struct {
	model.Document
	DocumentType documentTypeInfo   `json:"document_type"`
	Department   *departmentSummary `json:"department"`
	Year         *int               `json:"year"`
}
type documentRow struct {
	model.Document
	TypeName           string
	TypeCode           string
	TypeHasCorrelative bool
	DepartmentName     string
	DepartmentCode     string
	DepartmentActive   bool
	Year               *int
}

func documentQuery(db *gorm.DB) *gorm.DB {
	return db.Table("documents d").
		Joins("JOIN document_types dt ON dt.id=d.document_type_id").
		Joins("LEFT JOIN departments dep ON dep.id=d.department_id").
		Joins("LEFT JOIN document_sequences ds ON ds.id=d.document_number")
}

const documentFields = "d.*,dt.name AS type_name,dt.code AS type_code,dt.has_correlative AS type_has_correlative,dep.name AS department_name,dep.code AS department_code,dep.is_active AS department_active,ds.year"

func rowDocument(r documentRow) documentView {
	var dep *departmentSummary
	if r.DepartmentID != nil {
		dep = &departmentSummary{*r.DepartmentID, r.DepartmentName, r.DepartmentCode, r.DepartmentActive}
	}
	if r.CorrelativeNumber != nil && dep != nil {
		formatted := formatDocumentNumber(*r.CorrelativeNumber, dep.Code)
		r.DocumentNumber = &formatted
	}
	return documentView{r.Document, documentTypeInfo{r.DocumentTypeID, r.TypeName, r.TypeCode, r.TypeHasCorrelative}, dep, r.Year}
}
func (a *App) listDocuments(c *gin.Context) {
	p, ok := directoryQuery(c)
	if !ok {
		return
	}
	order, ok := documentSort(c, map[string]string{"id": "d.id", "name": "d.name", "created_at": "d.created_at", "updated_at": "d.updated_at", "correlative_number": "d.correlative_number", "document_type": "dt.name", "department": "dep.name", "year": "ds.year"}, "created_at", "desc")
	if !ok {
		return
	}
	q := documentQuery(a.DB.WithContext(c.Request.Context()))
	for _, field := range []string{"document_type_id", "department_id", "created_by"} {
		if raw, exists := c.GetQuery(field); exists {
			value, err := parseID(raw)
			if err != nil {
				fail(c, 400, errors.New("invalid "+field))
				return
			}
			q = q.Where("d."+field+"=?", value)
		}
	}
	if raw, exists := c.GetQuery("year"); exists {
		year, err := strconv.Atoi(raw)
		if err != nil || year < 1 || year > 9999 {
			fail(c, 400, errors.New("invalid year"))
			return
		}
		q = q.Where("ds.year=?", year)
	}
	if s := strings.TrimSpace(c.Query("q")); s != "" {
		pattern := "%" + s + "%"
		q = q.Where("(d.name ILIKE ? OR d.description ILIKE ? OR dt.name ILIKE ? OR dt.code ILIKE ?)", pattern, pattern, pattern, pattern)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		accountError(c, err)
		return
	}
	var rows []documentRow
	if err := q.Select(documentFields).Order(order).Offset((p.Page - 1) * p.Size).Limit(p.Size).Scan(&rows).Error; err != nil {
		accountError(c, err)
		return
	}
	out := make([]documentView, 0, len(rows))
	for _, r := range rows {
		out = append(out, rowDocument(r))
	}
	directoryResult(c, out, total, p)
}
func (a *App) getDocument(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, errors.New("invalid document id"))
		return
	}
	var rows []documentRow
	if err := documentQuery(a.DB.WithContext(c.Request.Context())).Select(documentFields).Where("d.id=?", id).Limit(1).Scan(&rows).Error; err != nil {
		accountError(c, err)
		return
	}
	if len(rows) == 0 {
		fail(c, 404, errors.New("document not found"))
		return
	}
	c.JSON(200, rowDocument(rows[0]))
}
