package app

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"std/internal/model"
	"strconv"
	"strings"
	"time"
)

type App struct {
	DB *gorm.DB
}
type recipientInput struct {
	RecipientID  *int64 `json:"recipient_id"`
	DepartmentID *int64 `json:"department_id"`
	Type         string `json:"type"`
}
type messageInput struct {
	CreatedBy    int64            `json:"created_by"`
	DepartmentID *int64           `json:"sender_department_id"`
	ProcedureID  *int64           `json:"procedure_id"`
	MessageType  string           `json:"message_type"`
	Visibility   string           `json:"visibility"`
	Status       string           `json:"status"`
	Subject      string           `json:"subject"`
	Body         string           `json:"body"`
	Recipients   []recipientInput `json:"recipients"`
	DocumentIDs  []int64          `json:"document_ids"`
}

func (a *App) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	registerSwagger(r)
	r.POST("/api/v1/departments", a.createDepartment)
	r.POST("/api/v1/users", a.createUser)
	r.POST("/api/v1/login", a.login)
	r.POST("/api/v1/documents", a.createDocument)
	r.GET("/api/v1/document-types", a.documentTypes)
	r.GET("/api/v1/documents", a.listDocuments)
	r.GET("/api/v1/documents/:id", a.getDocument)
	r.GET("/api/v1/users", a.listUsers)
	r.GET("/api/v1/departments", a.listDepartments)
	r.GET("/api/v1/departments/organization-chart", a.organizationChart)
	v := r.Group("/api/v1")
	v.GET("/departments/:id", a.department)
	v.POST("/document-types", a.createDocumentType)
	v.PATCH("/document-types/:id", a.patchDocumentType)
	v.POST("/procedures", a.createProcedure)
	v.GET("/procedures", a.procedures)
	v.GET("/procedures/by-sender-department/:id", a.proceduresBySenderDepartment)
	v.GET("/procedures/by-sender-user/:id", a.proceduresBySenderUser)
	v.GET("/procedures/:id", a.procedure)
	v.GET("/procedures/:id/messages", a.procedureMessages)
	v.POST("/procedures/:id/close", a.closeProcedure)
	v.POST("/messages", a.createMessage(nil, ""))
	v.GET("/messages/by-sender-department/:id", a.messagesBySenderDepartment)
	v.GET("/messages/by-sender-user/:id", a.messagesBySenderUser)
	v.GET("/messages/:id", a.message)
	v.GET("/messages/:id/history", a.messageHistory)
	v.POST("/messages/:id/reply", a.relatedMessage("REPLY"))
	v.POST("/messages/:id/forward", a.relatedMessage("FORWARD"))
	v.GET("/messages/inbox/personal", a.personalInbox)
	v.GET("/messages/inbox/department", a.departmentInbox)
	v.GET("/messages/sent", a.sent)
	v.GET("/message-recipients", a.unreadRecipients)
	v.POST("/message-recipients/:id/read", a.readRecipient)
	v.PATCH("/message-recipients/:id/status", a.updateRecipientStatus)
	v.GET("/messages/:id/documents", a.messageDocuments)
	return r
}
func fail(c *gin.Context, status int, err error) { c.JSON(status, gin.H{"error": err.Error()}) }
func parseID(raw string) (int64, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		return 0, errors.New("id must be a positive integer")
	}
	return value, nil
}

// id devuelve el usuario que hace la peticion, enviado como query param ?user_id=.
func id(c *gin.Context) (int64, bool) {
	u, e := parseID(c.Query("user_id"))
	if e != nil {
		return 0, false
	}
	return u, true
}

// member indica si el usuario pertenece activamente al area. MVP: no se distinguen roles.
func (a *App) member(user, dep int64) bool {
	var n int64
	a.DB.Model(&model.DepartmentMember{}).Where("user_id=? and department_id=? and is_active", user, dep).Count(&n)
	return n > 0
}
func (a *App) accessible(m model.Message, user int64) bool {
	if m.CreatedBy == user {
		return true
	}
	var n int64
	a.DB.Model(&model.MessageRecipient{}).Where("message_id=? and recipient_id=?", m.ID, user).Count(&n)
	if n > 0 {
		return true
	}
	if m.Visibility == "DEPARTMENT" && m.DepartmentID != nil && a.member(user, *m.DepartmentID) {
		return true
	}
	a.DB.Table("message_recipients mr join department_members dm on dm.department_id=mr.department_id").Where("mr.message_id=? and dm.user_id=? and dm.is_active", m.ID, user).Count(&n)
	return n > 0
}
func validateMessage(in messageInput) error {
	if in.Status == "" {
		in.Status = "HECHO"
	}
	if in.Status != "BORRADOR" && in.Status != "HECHO" {
		return errors.New("status must be BORRADOR or HECHO")
	}
	if in.MessageType == "DEPARTMENT" && in.DepartmentID == nil {
		return errors.New("DEPARTMENT requires department_id")
	}
	if in.MessageType != "PERSONAL" && in.MessageType != "DEPARTMENT" {
		return errors.New("invalid message_type")
	}
	if in.Visibility != "PRIVATE" && in.Visibility != "DEPARTMENT" {
		return errors.New("invalid visibility")
	}
	if in.Visibility == "DEPARTMENT" && in.MessageType != "DEPARTMENT" {
		return errors.New("DEPARTMENT visibility requires DEPARTMENT message_type")
	}
	if in.Status == "HECHO" && (strings.TrimSpace(in.Subject) == "" || strings.TrimSpace(in.Body) == "") {
		return errors.New("subject and body are required")
	}
	to := 0
	seen := map[string]bool{}
	for _, r := range in.Recipients {
		if (r.RecipientID == nil) == (r.DepartmentID == nil) {
			return errors.New("each recipient must be user xor department")
		}
		if r.Type != "TO" && r.Type != "CC" {
			return errors.New("recipient type must be TO or CC")
		}
		if r.Type == "TO" {
			to++
		}
		key := fmt.Sprintf("%v/%v", r.RecipientID, r.DepartmentID)
		if seen[key] {
			return errors.New("duplicate recipient")
		}
		seen[key] = true
	}
	if in.Status == "HECHO" && to == 0 {
		return errors.New("at least one TO recipient is required")
	}
	return nil
}
func (a *App) createMessage(relation *int64, relationType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in messageInput
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, errors.New("invalid json"))
			return
		}
		if in.Status == "" {
			in.Status = "HECHO"
		}
		user := in.CreatedBy
		if user < 1 {
			fail(c, 400, errors.New("created_by is required"))
			return
		}
		if e := validateMessage(in); e != nil {
			fail(c, 400, e)
			return
		}
		if in.DepartmentID != nil && !a.member(user, *in.DepartmentID) {
			fail(c, 403, errors.New("not authorized to represent department"))
			return
		}
		e := a.DB.Transaction(func(tx *gorm.DB) error {
			if in.ProcedureID != nil {
				var p model.Procedure
				if e := tx.First(&p, "id=?", *in.ProcedureID).Error; e != nil {
					return errors.New("procedure not found")
				}
				if p.Status != "OPEN" {
					return errors.New("procedure is closed")
				}
				if p.CreatedBy != user && (p.DepartmentID == nil || !a.member(user, *p.DepartmentID)) {
					return errors.New("no access to procedure")
				}
			}
			var sentAt *time.Time
			if in.Status == "HECHO" {
				now := time.Now()
				sentAt = &now
			}
			var senderUserID *int64
			if in.MessageType == "PERSONAL" {
				senderUserID = &user
			}
			m := model.Message{CreatedBy: user, SenderUserID: senderUserID, DepartmentID: in.DepartmentID, ProcedureID: in.ProcedureID, MessageType: in.MessageType, Visibility: in.Visibility, Status: in.Status, Subject: in.Subject, Body: in.Body, SentAt: sentAt}
			if e := tx.Create(&m).Error; e != nil {
				return e
			}
			for _, r := range in.Recipients {
				if e := tx.Create(&model.MessageRecipient{MessageID: m.ID, RecipientID: r.RecipientID, DepartmentID: r.DepartmentID, Type: r.Type, Status: "PENDING"}).Error; e != nil {
					return e
				}
			}
			for _, d := range in.DocumentIDs {
				if e := tx.Create(&model.MessageDocument{MessageID: m.ID, DocumentID: d}).Error; e != nil {
					return e
				}
			}
			if relation != nil {
				if *relation == m.ID {
					return errors.New("self relation")
				}
				if e := tx.Create(&model.MessageRelation{SourceMessageID: m.ID, TargetMessageID: *relation, RelationType: relationType, CreatedBy: user}).Error; e != nil {
					return e
				}
			}
			action := map[string]string{"": "MESSAGE_SENT", "REPLY": "MESSAGE_REPLIED", "FORWARD": "MESSAGE_FORWARDED"}[relationType]
			if in.Status == "BORRADOR" {
				action = "MESSAGE_DRAFT_SAVED"
			}
			return tx.Create(&model.AuditEvent{ActorID: user, EntityType: "MESSAGE", EntityID: m.ID, Action: action, Metadata: "{}"}).Error
		})
		if e != nil {
			fail(c, 400, e)
			return
		}
		c.JSON(201, gin.H{"status": "created"})
	}
}
func (a *App) relatedMessage(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		mid, e := parseID(c.Param("id"))
		if e != nil {
			fail(c, 400, e)
			return
		}
		var original model.Message
		if a.DB.First(&original, "id=?", mid).Error != nil {
			fail(c, 404, errors.New("message not found"))
			return
		}
		var input messageInput
		if c.ShouldBindJSON(&input) != nil {
			fail(c, 400, errors.New("invalid json"))
			return
		}
		if input.Status == "" {
			input.Status = "HECHO"
		}
		if !a.accessible(original, input.CreatedBy) {
			fail(c, 403, errors.New("message access denied"))
			return
		}
		if input.ProcedureID == nil {
			input.ProcedureID = original.ProcedureID
		}
		c.Set("_body", input)
		a.createMessageWithInput(c, &mid, kind, input)
	}
}
func (a *App) createMessageWithInput(c *gin.Context, rel *int64, kind string, in messageInput) { // duplicate-free adapter through local transaction
	if in.Status == "" {
		in.Status = "HECHO"
	}
	u := in.CreatedBy
	if u < 1 {
		fail(c, 400, errors.New("created_by is required"))
		return
	}
	if e := validateMessage(in); e != nil {
		fail(c, 400, e)
		return
	}
	if in.DepartmentID != nil && !a.member(u, *in.DepartmentID) {
		fail(c, 403, errors.New("not authorized to represent department"))
		return
	}
	e := a.DB.Transaction(func(tx *gorm.DB) error {
		var sentAt *time.Time
		if in.Status == "HECHO" {
			now := time.Now()
			sentAt = &now
		}
		var senderUserID *int64
		if in.MessageType == "PERSONAL" {
			senderUserID = &u
		}
		m := model.Message{CreatedBy: u, SenderUserID: senderUserID, DepartmentID: in.DepartmentID, ProcedureID: in.ProcedureID, MessageType: in.MessageType, Visibility: in.Visibility, Status: in.Status, Subject: in.Subject, Body: in.Body, SentAt: sentAt}
		if e := tx.Create(&m).Error; e != nil {
			return e
		}
		for _, r := range in.Recipients {
			if e := tx.Create(&model.MessageRecipient{MessageID: m.ID, RecipientID: r.RecipientID, DepartmentID: r.DepartmentID, Type: r.Type, Status: "PENDING"}).Error; e != nil {
				return e
			}
		}
		for _, d := range in.DocumentIDs {
			if e := tx.Create(&model.MessageDocument{MessageID: m.ID, DocumentID: d}).Error; e != nil {
				return e
			}
		}
		if e := tx.Create(&model.MessageRelation{SourceMessageID: m.ID, TargetMessageID: *rel, RelationType: kind, CreatedBy: u}).Error; e != nil {
			return e
		}
		action := "MESSAGE_" + kind
		if in.Status == "BORRADOR" {
			action = "MESSAGE_DRAFT_SAVED"
		}
		return tx.Create(&model.AuditEvent{ActorID: u, EntityType: "MESSAGE", EntityID: m.ID, Action: action, Metadata: "{}"}).Error
	})
	if e != nil {
		fail(c, 400, e)
		return
	}
	c.JSON(201, gin.H{"status": "created"})
}

func page(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	s, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if p < 1 {
		p = 1
	}
	if s < 1 || s > 100 {
		s = 20
	}
	return p, s
}
func (a *App) list(c *gin.Context, q *gorm.DB) {
	p, s := page(c)
	rows := make([]model.Message, 0)
	var total int64
	q.Count(&total)
	q.Order("messages.created_at desc").Offset((p - 1) * s).Limit(s).Find(&rows)
	c.JSON(200, gin.H{"data": rows, "page": p, "page_size": s, "total": total})
}
func (a *App) departments(c *gin.Context) {
	x := make([]model.Department, 0)
	a.DB.Where("is_active").Order("name").Find(&x)
	c.JSON(200, gin.H{"data": x})
}
func (a *App) department(c *gin.Context) {
	var x model.Department
	if a.DB.First(&x, "id=?", c.Param("id")).Error != nil {
		fail(c, 404, errors.New("department not found"))
		return
	}
	c.JSON(200, x)
}
func (a *App) documentTypes(c *gin.Context) {
	a.listDocumentTypes(c)
}
func (a *App) createDocumentType(c *gin.Context) {
	var x model.DocumentType
	if c.ShouldBindJSON(&x) != nil || strings.TrimSpace(x.Name) == "" {
		fail(c, 400, errors.New("name is required"))
		return
	}
	if e := a.DB.Create(&x).Error; e != nil {
		fail(c, 400, e)
		return
	}
	c.JSON(201, x)
}
func (a *App) patchDocumentType(c *gin.Context) {
	var x model.DocumentType
	if a.DB.First(&x, "id=?", c.Param("id")).Error != nil {
		fail(c, 404, errors.New("document type not found"))
		return
	}
	var in struct {
		Name     *string `json:"name"`
		Code     *string `json:"code"`
		IsActive *bool   `json:"is_active"`
	}
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, errors.New("invalid json"))
		return
	}
	a.DB.Model(&x).Updates(in)
	c.JSON(200, x)
}
func (a *App) createDocument(c *gin.Context) {
	var in CreateDocumentInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, errInput)
		return
	}
	d, err := a.CreateDocument(c.Request.Context(), in)
	if errors.Is(err, errDocumentInput) {
		fail(c, 400, err)
		return
	}
	if errors.Is(err, errDocumentMembership) {
		fail(c, 403, err)
		return
	}
	if err != nil {
		accountError(c, err)
		return
	}
	c.JSON(201, d)
}
func (a *App) documents(c *gin.Context) {
	p, s := page(c)
	var x []model.Document
	a.DB.Offset((p - 1) * s).Limit(s).Order("created_at desc").Find(&x)
	c.JSON(200, gin.H{"data": x, "page": p, "page_size": s})
}
func (a *App) document(c *gin.Context) {
	var x model.Document
	if a.DB.First(&x, "id=?", c.Param("id")).Error != nil {
		fail(c, 404, errors.New("document not found"))
		return
	}
	c.JSON(200, x)
}
func (a *App) createProcedure(c *gin.Context) {
	var in struct {
		Name          string  `json:"name"`
		Description   *string `json:"description"`
		ProcedureType string  `json:"procedure_type"`
		CreatedBy     int64   `json:"created_by"`
		DepartmentID  *int64  `json:"sender_department_id"`
	}
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Name) == "" || in.CreatedBy < 1 || (in.ProcedureType != "PERSONAL" && in.ProcedureType != "DEPARTMENT") || (in.ProcedureType == "DEPARTMENT" && in.DepartmentID == nil) {
		fail(c, 400, errors.New("name is required"))
		return
	}
	u := in.CreatedBy
	if in.DepartmentID != nil && !a.member(u, *in.DepartmentID) {
		fail(c, 403, errors.New("not authorized for department"))
		return
	}
	var senderUserID *int64
	if in.ProcedureType == "PERSONAL" {
		senderUserID = &u
	}
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		in.Description = &d
		if d == "" {
			in.Description = nil
		}
	}
	p := model.Procedure{Name: strings.TrimSpace(in.Name), Description: in.Description, ProcedureType: in.ProcedureType, CreatedBy: u, SenderUserID: senderUserID, DepartmentID: in.DepartmentID, Status: "OPEN"}
	if e := a.DB.Transaction(func(tx *gorm.DB) error {
		number, e := nextProcedureNumber(tx, p)
		if e != nil {
			return e
		}
		p.ProcedureNumber = &number
		if e := tx.Create(&p).Error; e != nil {
			return e
		}
		return tx.Create(&model.AuditEvent{ActorID: u, EntityType: "PROCEDURE", EntityID: p.ID, Action: "PROCEDURE_CREATED", Metadata: "{}"}).Error
	}); e != nil {
		fail(c, 400, e)
		return
	}
	c.JSON(201, p)
}
func (a *App) procedureAllowed(p model.Procedure, u int64) bool {
	return p.CreatedBy == u || (p.DepartmentID != nil && a.member(u, *p.DepartmentID))
}
func (a *App) procedures(c *gin.Context) {
	u, _ := id(c)
	q := a.DB.Model(&model.Procedure{}).Where("created_by=?", u).Or("department_id in (?)", a.DB.Model(&model.DepartmentMember{}).Select("department_id").Where("user_id=? and is_active", u))
	if s := c.Query("status"); s != "" {
		q = q.Where("status=?", s)
	}
	if x := c.Query("q"); x != "" {
		q = q.Where("(name ILIKE ? OR procedure_number ILIKE ?)", "%"+x+"%", "%"+x+"%")
	}
	p, z := page(c)
	rows := make([]model.Procedure, 0)
	var total int64
	q.Count(&total)
	q.Order("created_at desc").Offset((p - 1) * z).Limit(z).Find(&rows)
	c.JSON(200, gin.H{"data": rows, "total": total, "page": p})
}
func procedureFilters(c *gin.Context, q *gorm.DB) *gorm.DB {
	if status := c.Query("status"); status != "" {
		q = q.Where("status=?", status)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("(name ILIKE ? OR procedure_number ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	return q
}
func procedurePage(c *gin.Context, q *gorm.DB) {
	p, size := page(c)
	q = procedureFilters(c, q)
	var total int64
	q.Count(&total)
	rows := make([]model.Procedure, 0)
	q.Order("created_at desc").Offset((p - 1) * size).Limit(size).Find(&rows)
	c.JSON(200, gin.H{"data": rows, "total": total, "page": p, "page_size": size})
}
func (a *App) proceduresBySenderDepartment(c *gin.Context) {
	departmentID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, errors.New("invalid sender department id"))
		return
	}
	u, _ := id(c)
	if !a.member(u, departmentID) {
		fail(c, 403, errors.New("procedure access denied"))
		return
	}
	procedurePage(c, a.DB.Model(&model.Procedure{}).Where("sender_department_id=?", departmentID))
}
func (a *App) proceduresBySenderUser(c *gin.Context) {
	senderUserID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, errors.New("invalid sender user id"))
		return
	}
	u, _ := id(c)
	if u != senderUserID {
		fail(c, 403, errors.New("procedure access denied"))
		return
	}
	procedurePage(c, a.DB.Model(&model.Procedure{}).Where("sender_user_id=?", senderUserID))
}
func (a *App) procedure(c *gin.Context) {
	u, _ := id(c)
	var p model.Procedure
	if a.DB.First(&p, "id=?", c.Param("id")).Error != nil {
		fail(c, 404, errors.New("procedure not found"))
		return
	}
	if !a.procedureAllowed(p, u) {
		fail(c, 403, errors.New("procedure access denied"))
		return
	}
	allMessages := make([]model.Message, 0)
	a.DB.Where("procedure_id=?", p.ID).Order("created_at asc").Find(&allMessages)
	messages := make([]model.Message, 0, len(allMessages))
	messageIDs := make([]int64, 0, len(allMessages))
	visible := make(map[int64]bool, len(allMessages))
	for _, message := range allMessages {
		if a.accessible(message, u) {
			messages = append(messages, message)
			messageIDs = append(messageIDs, message.ID)
			visible[message.ID] = true
		}
	}
	recipients := make([]model.MessageRecipient, 0)
	relations := make([]model.MessageRelation, 0)
	if len(messageIDs) > 0 {
		a.DB.Where("message_id in ?", messageIDs).Order("created_at asc").Find(&recipients)
		allRelations := make([]model.MessageRelation, 0)
		a.DB.Where("source_message_id in ? or target_message_id in ?", messageIDs, messageIDs).Order("created_at asc").Find(&allRelations)
		for _, relation := range allRelations {
			if visible[relation.SourceMessageID] && visible[relation.TargetMessageID] {
				relations = append(relations, relation)
			}
		}
	}
	c.JSON(200, gin.H{
		"procedure":          p,
		"messages":           messages,
		"message_recipients": recipients,
		"message_relations":  relations,
	})
}
func (a *App) procedureMessages(c *gin.Context) {
	u, _ := id(c)
	var p model.Procedure
	if a.DB.First(&p, "id=?", c.Param("id")).Error != nil {
		fail(c, 404, errors.New("procedure not found"))
		return
	}
	if !a.procedureAllowed(p, u) {
		fail(c, 403, errors.New("procedure access denied"))
		return
	}
	q := a.DB.Model(&model.Message{}).Where("procedure_id=?", p.ID)
	pno, s := page(c)
	all := make([]model.Message, 0)
	q.Order("created_at desc").Offset((pno - 1) * s).Limit(s).Find(&all)
	out := make([]model.Message, 0)
	for _, m := range all {
		if a.accessible(m, u) {
			out = append(out, m)
		}
	}
	c.JSON(200, gin.H{"data": out, "page": pno, "page_size": s})
}
func (a *App) closeProcedure(c *gin.Context) {
	u, _ := id(c)
	pid, e := parseID(c.Param("id"))
	if e != nil {
		fail(c, 400, e)
		return
	}
	var in struct {
		ClosingMessageID *int64 `json:"closing_message_id"`
		ClosingNote      string `json:"closing_note"`
	}
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.ClosingNote) == "" {
		fail(c, 400, errors.New("closing_note is required"))
		return
	}
	var p model.Procedure
	e = a.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id=?", pid).Error; e != nil {
			return errors.New("procedure not found")
		}
		if p.Status != "OPEN" {
			return errors.New("procedure already closed")
		}
		if p.CreatedBy != u && (p.DepartmentID == nil || !a.member(u, *p.DepartmentID)) {
			return errors.New("close denied")
		}
		if in.ClosingMessageID != nil {
			var n int64
			tx.Model(&model.Message{}).Where("id=? and procedure_id=?", *in.ClosingMessageID, p.ID).Count(&n)
			if n != 1 {
				return errors.New("closing message does not belong to procedure")
			}
		}
		now := time.Now()
		p.Status = "CLOSED"
		p.ClosedBy = &u
		p.ClosedAt = &now
		p.ClosingMessageID = in.ClosingMessageID
		p.ClosingNote = &in.ClosingNote
		if e := tx.Save(&p).Error; e != nil {
			return e
		}
		return tx.Create(&model.AuditEvent{ActorID: u, EntityType: "PROCEDURE", EntityID: p.ID, Action: "PROCEDURE_CLOSED", Metadata: "{}"}).Error
	})
	if e != nil {
		fail(c, 400, e)
		return
	}
	c.JSON(200, p)
}
func (a *App) message(c *gin.Context) {
	u, _ := id(c)
	var m model.Message
	if a.DB.First(&m, "id=?", c.Param("id")).Error != nil {
		fail(c, 404, errors.New("message not found"))
		return
	}
	if !a.accessible(m, u) {
		fail(c, 403, errors.New("message access denied"))
		return
	}
	c.JSON(200, m)
}

type unreadRecipient struct {
	ID           int64     `json:"id"`
	MessageID    int64     `json:"message_id"`
	RecipientID  *int64    `json:"recipient_id"`
	DepartmentID *int64    `json:"department_id"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Subject      string    `json:"subject"`
	CreatedAt    time.Time `json:"created_at"`
}

// unreadRecipients returns only recipient records that the requesting user has not read.
func (a *App) unreadRecipients(c *gin.Context) {
	u, ok := id(c)
	if !ok {
		fail(c, 400, errors.New("user_id is required"))
		return
	}
	p, size := page(c)
	query := a.DB.Table("message_recipients mr").
		Select("mr.id, mr.message_id, mr.recipient_id, mr.department_id, mr.type, 'UNREAD' AS status, m.subject, m.created_at").
		Joins("join messages m on m.id=mr.message_id").
		Joins("left join message_recipient_reads mrr on mrr.message_recipient_id=mr.id and mrr.user_id=?", u).
		Joins("left join department_members dm on dm.department_id=mr.department_id and dm.user_id=? and dm.is_active", u).
		Where("m.status='HECHO' and (mr.recipient_id=? or dm.user_id is not null)", u).
		Where("(mr.recipient_id=? and mr.read_at is null) or (mr.department_id is not null and mrr.id is null)", u)
	var total int64
	query.Count(&total)
	rows := make([]unreadRecipient, 0)
	if err := query.Order("m.created_at desc").Offset((p - 1) * size).Limit(size).Scan(&rows).Error; err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(200, gin.H{"data": rows, "total": total, "page": p, "page_size": size})
}

// messageHistory follows the complete graph of message_relations from a message.
func (a *App) messageHistory(c *gin.Context) {
	u, ok := id(c)
	if !ok {
		fail(c, 400, errors.New("user_id is required"))
		return
	}
	mid, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, err)
		return
	}
	var root model.Message
	if a.DB.First(&root, "id=?", mid).Error != nil {
		fail(c, 404, errors.New("message not found"))
		return
	}
	if !a.accessible(root, u) {
		fail(c, 403, errors.New("message access denied"))
		return
	}
	var ids []int64
	graph := `WITH RECURSIVE connected(id) AS (
		SELECT CAST(? AS bigint)
		UNION
		SELECT CASE WHEN r.source_message_id = c.id THEN r.target_message_id ELSE r.source_message_id END
		FROM connected c
		JOIN message_relations r ON r.source_message_id = c.id OR r.target_message_id = c.id
	) SELECT id FROM connected`
	if err := a.DB.Raw(graph, mid).Scan(&ids).Error; err != nil {
		fail(c, 500, err)
		return
	}
	messages := make([]model.Message, 0)
	a.DB.Where("id in ?", ids).Order("created_at asc").Find(&messages)
	visible := make(map[int64]bool, len(messages))
	visibleMessages := make([]model.Message, 0, len(messages))
	for _, message := range messages {
		if a.accessible(message, u) {
			visible[message.ID] = true
			visibleMessages = append(visibleMessages, message)
		}
	}
	relations := make([]model.MessageRelation, 0)
	a.DB.Where("source_message_id in ? and target_message_id in ?", ids, ids).Order("created_at asc").Find(&relations)
	visibleRelations := make([]model.MessageRelation, 0, len(relations))
	for _, relation := range relations {
		if visible[relation.SourceMessageID] && visible[relation.TargetMessageID] {
			visibleRelations = append(visibleRelations, relation)
		}
	}
	c.JSON(200, gin.H{"message_id": mid, "messages": visibleMessages, "relations": visibleRelations})
}
func messagePage(c *gin.Context, q *gorm.DB) {
	if value := c.Query("message_type"); value != "" {
		q = q.Where("message_type=?", value)
	}
	if value := c.Query("visibility"); value != "" {
		q = q.Where("visibility=?", value)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("subject ILIKE ? OR body ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if raw := c.Query("procedure_id"); raw != "" {
		procedureID, err := parseID(raw)
		if err != nil {
			fail(c, 400, errors.New("invalid procedure_id"))
			return
		}
		q = q.Where("procedure_id=?", procedureID)
	}
	p, size := page(c)
	var total int64
	q.Count(&total)
	rows := make([]model.Message, 0)
	q.Order("created_at desc").Offset((p - 1) * size).Limit(size).Find(&rows)
	c.JSON(200, gin.H{"data": rows, "total": total, "page": p, "page_size": size})
}
func (a *App) messagesBySenderDepartment(c *gin.Context) {
	departmentID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, errors.New("invalid sender department id"))
		return
	}
	u, _ := id(c)
	if !a.member(u, departmentID) {
		fail(c, 403, errors.New("message access denied"))
		return
	}
	messagePage(c, a.DB.Model(&model.Message{}).Where("sender_department_id=?", departmentID))
}
func (a *App) messagesBySenderUser(c *gin.Context) {
	senderUserID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, errors.New("invalid sender user id"))
		return
	}
	u, _ := id(c)
	if u != senderUserID {
		fail(c, 403, errors.New("message access denied"))
		return
	}
	messagePage(c, a.DB.Model(&model.Message{}).Where("sender_user_id=?", senderUserID))
}
func (a *App) personalInbox(c *gin.Context) {
	u, _ := id(c)
	a.list(c, a.DB.Model(&model.Message{}).Joins("join message_recipients mr on mr.message_id=messages.id").Where("mr.recipient_id=? and messages.status='HECHO'", u))
}
func (a *App) departmentInbox(c *gin.Context) {
	u, _ := id(c)
	a.list(c, a.DB.Model(&model.Message{}).Joins("join message_recipients mr on mr.message_id=messages.id join department_members dm on dm.department_id=mr.department_id").Where("dm.user_id=? and dm.is_active and messages.status='HECHO'", u))
}
func (a *App) sent(c *gin.Context) {
	u, _ := id(c)
	a.list(c, a.DB.Model(&model.Message{}).Where("created_by=? and status='HECHO'", u))
}
func (a *App) readRecipient(c *gin.Context) {
	u, _ := id(c)
	rid, e := parseID(c.Param("id"))
	if e != nil {
		fail(c, 400, e)
		return
	}
	var r model.MessageRecipient
	if a.DB.First(&r, "id=?", rid).Error != nil {
		fail(c, 404, errors.New("recipient not found"))
		return
	}
	allowed := r.RecipientID != nil && *r.RecipientID == u || r.DepartmentID != nil && a.member(u, *r.DepartmentID)
	if !allowed {
		fail(c, 403, errors.New("read denied"))
		return
	}
	now := time.Now()
	e = a.DB.Transaction(func(tx *gorm.DB) error {
		if r.RecipientID != nil {
			r.Status = "READ"
			r.ReadAt = &now
			if e := tx.Save(&r).Error; e != nil {
				return e
			}
		} else {
			if e := tx.Where("message_recipient_id=? and user_id=?", r.ID, u).FirstOrCreate(&model.MessageRecipientRead{MessageRecipientID: r.ID, UserID: u, ReadAt: now}).Error; e != nil {
				return e
			}
		}
		return tx.Create(&model.AuditEvent{ActorID: u, EntityType: "MESSAGE_RECIPIENT", EntityID: r.ID, Action: "MESSAGE_READ", Metadata: "{}"}).Error
	})
	if e != nil {
		fail(c, 400, e)
		return
	}
	c.Status(http.StatusNoContent)
}
func (a *App) updateRecipientStatus(c *gin.Context) {
	u, _ := id(c)
	rid, e := parseID(c.Param("id"))
	if e != nil {
		fail(c, 400, e)
		return
	}
	var r model.MessageRecipient
	if a.DB.First(&r, "id=?", rid).Error != nil {
		fail(c, 404, errors.New("recipient not found"))
		return
	}
	if !((r.RecipientID != nil && *r.RecipientID == u) || (r.DepartmentID != nil && a.member(u, *r.DepartmentID))) {
		fail(c, 403, errors.New("status update denied"))
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&in) != nil || !map[string]bool{"PENDING": true, "READ": true, "IN_PROGRESS": true, "ANSWERED": true, "CLOSED": true}[in.Status] {
		fail(c, 400, errors.New("invalid status"))
		return
	}
	r.Status = in.Status
	if in.Status == "READ" {
		now := time.Now()
		r.ReadAt = &now
	}
	a.DB.Save(&r)
	c.JSON(200, r)
}
func (a *App) messageDocuments(c *gin.Context) {
	u, _ := id(c)
	mid, e := parseID(c.Param("id"))
	if e != nil {
		fail(c, 400, e)
		return
	}
	var m model.Message
	if a.DB.First(&m, "id=?", mid).Error != nil {
		fail(c, 404, errors.New("message not found"))
		return
	}
	if !a.accessible(m, u) {
		fail(c, 403, errors.New("message access denied"))
		return
	}
	p, s := page(c)
	var docs []model.Document
	a.DB.Table("documents d join message_documents md on md.document_id=d.id").Where("md.message_id=?", mid).Offset((p - 1) * s).Limit(s).Find(&docs)
	c.JSON(200, gin.H{"data": docs, "page": p, "page_size": s})
}
