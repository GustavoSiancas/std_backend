package app

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"std/internal/model"
	"strconv"
	"strings"
)

type directoryPage struct {
	Page int
	Size int
}

func directoryQuery(c *gin.Context) (directoryPage, bool) {
	p, e := strconv.Atoi(c.DefaultQuery("page", "1"))
	s, f := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if e != nil || f != nil || p < 1 || s < 1 || s > 100 || p > 1000000 {
		fail(c, 400, errors.New("page must be 1â€“1000000 and page_size 1â€“100"))
		return directoryPage{}, false
	}
	return directoryPage{p, s}, true
}
func boolFilter(c *gin.Context, q *gorm.DB, key, column string) (*gorm.DB, bool) {
	if raw, exists := c.GetQuery(key); exists {
		b, e := strconv.ParseBool(raw)
		if e != nil {
			fail(c, 400, errors.New(key+" must be true or false"))
			return q, false
		}
		q = q.Where(column+" = ?", b)
	}
	return q, true
}
func directoryResult(c *gin.Context, data any, total int64, p directoryPage) {
	pages := (total + int64(p.Size) - 1) / int64(p.Size)
	c.JSON(200, gin.H{"data": data, "page": p.Page, "page_size": p.Size, "total": total, "total_pages": pages})
}

type userArea struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Code             string `json:"code"`
	Role             string `json:"role"`
	IsActive         bool   `json:"is_active"`
	MembershipActive bool   `json:"membership_active"`
}
type directoryUser struct {
	model.User
	Departments []userArea `json:"departments"`
}

func (a *App) listUsers(c *gin.Context) {
	p, ok := directoryQuery(c)
	if !ok {
		return
	}
	db := a.DB.WithContext(c.Request.Context())
	q := db.Model(&model.User{})
	if raw, exists := c.GetQuery("department_id"); exists {
		dep, err := parseID(raw)
		if err != nil {
			fail(c, 400, errors.New("invalid department_id"))
			return
		}
		q = q.Where("EXISTS (SELECT 1 FROM department_members dm WHERE dm.user_id=users.id AND dm.department_id=? AND dm.is_active)", dep)
	}
	if search := strings.TrimSpace(c.Query("q")); search != "" {
		pattern := "%" + search + "%"
		q = q.Where("(name ILIKE ? OR username ILIKE ? OR email ILIKE ?)", pattern, pattern, pattern)
	}
	q, ok = boolFilter(c, q, "admin", "users.admin")
	if !ok {
		return
	}
	q, ok = boolFilter(c, q, "is_active", "users.is_active")
	if !ok {
		return
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		accountError(c, err)
		return
	}
	users := make([]model.User, 0)
	if err := q.Select("id,name,email,username,admin,is_active,created_at,updated_at").Order("name ASC, id ASC").Offset((p.Page - 1) * p.Size).Limit(p.Size).Find(&users).Error; err != nil {
		accountError(c, err)
		return
	}
	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	// Explicit scan fields avoid embedded unexported type reflection ambiguity.
	var rows []struct {
		UserID           int64
		ID               int64
		Name             string
		Code             string
		Role             string
		IsActive         bool
		MembershipActive bool
	}
	if len(ids) > 0 {
		err := db.Table("department_members dm").Select("dm.user_id,d.id,d.name,d.code,dm.role,d.is_active,dm.is_active AS membership_active").Joins("JOIN departments d ON d.id=dm.department_id").Where("dm.user_id IN ?", ids).Order("d.name ASC,d.id ASC").Scan(&rows).Error
		if err != nil {
			accountError(c, err)
			return
		}
	}
	byUser := map[int64][]userArea{}
	for _, r := range rows {
		byUser[r.UserID] = append(byUser[r.UserID], userArea{r.ID, r.Name, r.Code, r.Role, r.IsActive, r.MembershipActive})
	}
	out := make([]directoryUser, 0, len(users))
	for _, u := range users {
		areas := byUser[u.ID]
		if areas == nil {
			areas = []userArea{}
		}
		out = append(out, directoryUser{u, areas})
	}
	directoryResult(c, out, total, p)
}
func (a *App) listDepartments(c *gin.Context) {
	p, ok := directoryQuery(c)
	if !ok {
		return
	}
	q := a.DB.WithContext(c.Request.Context()).Model(&model.Department{})
	if search := strings.TrimSpace(c.Query("q")); search != "" {
		pattern := "%" + search + "%"
		q = q.Where("(name ILIKE ? OR code ILIKE ?)", pattern, pattern)
	}
	q, ok = boolFilter(c, q, "is_active", "is_active")
	if !ok {
		return
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		accountError(c, err)
		return
	}
	rows := make([]model.Department, 0)
	if err := q.Order("name ASC,id ASC").Offset((p.Page - 1) * p.Size).Limit(p.Size).Find(&rows).Error; err != nil {
		accountError(c, err)
		return
	}
	ids := make([]int64, 0, len(rows))
	for _, d := range rows {
		if d.ParentID != nil {
			ids = append(ids, *d.ParentID)
		}
	}
	parents := make(map[int64]departmentSummary)
	if len(ids) > 0 {
		var loaded []model.Department
		if err := a.DB.WithContext(c.Request.Context()).Where("id IN ?", ids).Find(&loaded).Error; err != nil {
			accountError(c, err)
			return
		}
		for _, d := range loaded {
			parents[d.ID] = departmentSummary{d.ID, d.Name, d.Code, d.IsActive}
		}
	}
	out := make([]departmentWithParent, 0, len(rows))
	for _, d := range rows {
		var parent *departmentSummary
		if d.ParentID != nil {
			if found, ok := parents[*d.ParentID]; ok {
				copy := found
				parent = &copy
			}
		}
		out = append(out, departmentWithParent{d, parent})
	}
	directoryResult(c, out, total, p)
}
