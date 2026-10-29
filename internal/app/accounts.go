package app

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"std/internal/model"
)

type CreateDepartmentInput struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	ParentID *int64 `json:"parent_id"`
}
type CreateUserInput struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DepartmentID int64  `json:"department_id"`
}

var errInput = errors.New("invalid input")
var errCredentials = errors.New("invalid username or password")
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-login-password"), bcrypt.DefaultCost)

// CreateDepartment validates the parent before persisting an active area.
func (a *App) CreateDepartment(ctx context.Context, in CreateDepartmentInput) (model.Department, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	if in.Name == "" || len(in.Name) > 255 || in.Code == "" || len(in.Code) > 50 {
		return model.Department{}, errInput
	}
	db := a.DB.WithContext(ctx)
	if in.ParentID != nil {
		var n int64
		if err := db.Model(&model.Department{}).Where("id=? AND is_active", *in.ParentID).Count(&n).Error; err != nil {
			return model.Department{}, err
		}
		if n != 1 {
			return model.Department{}, errInput
		}
	}
	d := model.Department{Name: in.Name, Code: in.Code, ParentID: in.ParentID, IsActive: true}
	err := db.Create(&d).Error
	return d, err
}

// CreateUser stores the bcrypt hash and membership atomically.
func (a *App) CreateUser(ctx context.Context, in CreateUserInput) (model.User, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	email, err := mail.ParseAddress(in.Email)
	if err != nil || email.Address != in.Email || len(in.Email) > 255 || in.Name == "" || len(in.Name) > 255 || in.Username == "" || len(in.Username) > 100 || len(in.Password) < 8 || len(in.Password) > 72 || in.DepartmentID == 0 {
		return model.User{}, errInput
	}
	for _, r := range in.Username {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._-", r)) {
			return model.User{}, errInput
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	u := model.User{Name: in.Name, Email: in.Email, Username: in.Username, PasswordHash: hash, IsActive: true}
	err = a.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&model.Department{}).Where("id=? AND is_active", in.DepartmentID).Count(&n).Error; err != nil {
			return err
		}
		if n != 1 {
			return errInput
		}
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		return tx.Create(&model.DepartmentMember{DepartmentID: in.DepartmentID, UserID: u.ID, Role: "MEMBER", IsActive: true}).Error
	})
	return u, err
}

// Login checks credentials only; it does not issue a token or create a session.
func (a *App) Login(ctx context.Context, username, password string) (model.User, error) {
	var u model.User
	err := a.DB.WithContext(ctx).Where("lower(username)=?", strings.ToLower(strings.TrimSpace(username))).First(&u).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, err
	}
	hash := u.PasswordHash
	if len(hash) == 0 {
		hash = dummyHash
	}
	valid := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	if err != nil || !u.IsActive || !valid || len(u.PasswordHash) == 0 {
		return model.User{}, errCredentials
	}
	return u, nil
}
func accountError(c *gin.Context, err error) {
	var pg *pgconn.PgError
	switch {
	case errors.Is(err, errInput):
		fail(c, 400, errors.New("invalid fields or inactive/missing department; password must be 8â€“72 bytes"))
	case errors.Is(err, errCredentials):
		fail(c, 401, errCredentials)
	case errors.As(err, &pg) && pg.Code == "23505":
		fail(c, 409, errors.New("username, email or department code already exists"))
	case errors.As(err, &pg) && pg.Code == "23503":
		fail(c, 400, errors.New("referenced department does not exist"))
	default:
		fail(c, 500, errors.New("operation could not be completed"))
	}
}
func (a *App) createDepartment(c *gin.Context) {
	var in CreateDepartmentInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, errInput)
		return
	}
	d, err := a.CreateDepartment(c.Request.Context(), in)
	if err != nil {
		accountError(c, err)
		return
	}
	c.JSON(201, d)
}
func (a *App) createUser(c *gin.Context) {
	var in CreateUserInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, errInput)
		return
	}
	u, err := a.CreateUser(c.Request.Context(), in)
	if err != nil {
		accountError(c, err)
		return
	}
	c.JSON(201, gin.H{"user": u, "department_id": in.DepartmentID})
}
func (a *App) login(c *gin.Context) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&in) != nil || in.Username == "" || in.Password == "" {
		fail(c, 400, errInput)
		return
	}
	u, err := a.Login(c.Request.Context(), in.Username, in.Password)
	if err != nil {
		accountError(c, err)
		return
	}
	departments := make([]userArea, 0)
	err = a.DB.WithContext(c.Request.Context()).Table("department_members dm").
		Select("d.id,d.name,d.code,dm.role,d.is_active,dm.is_active AS membership_active").
		Joins("JOIN departments d ON d.id=dm.department_id").
		Where("dm.user_id=? AND dm.is_active AND d.is_active", u.ID).
		Order("d.name ASC,d.id ASC").Scan(&departments).Error
	if err != nil {
		accountError(c, err)
		return
	}
	c.JSON(200, gin.H{"user": directoryUser{User: u, Departments: departments}})
}
