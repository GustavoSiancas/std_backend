package app

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"os"
	"std/internal/model"
	"std/migrations"
	"strconv"
	"strings"
	"testing"
)

func TestAccountsHTTP(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	schema := "accounts_test_" + "99999999"[:8]
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema + ",public").Error; err != nil {
		t.Fatal(err)
	}
	if err := migrations.Apply(tx); err != nil {
		t.Fatal(err)
	}
	a := &App{DB: tx}
	router := a.Router()
	request := func(path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s: got %d expected %d: %s", path, w.Code, status, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "password") && status < 300 {
			t.Fatal("password field exposed")
		}
		return w
	}
	w := request("/api/v1/departments", `{"name":"Sistemas","code":"SIS"}`, 201)
	var dep model.Department
	if err := json.Unmarshal(w.Body.Bytes(), &dep); err != nil || dep.ID == 0 {
		t.Fatal("missing department id")
	}
	body := `{"name":"Maria","email":"maria@example.test","username":"Maria","password":"Example123!","department_id":` + strconv.FormatInt(dep.ID, 10) + `}`
	request("/api/v1/users", body, 201)
	var user model.User
	if err := tx.Where("username='maria'").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword(user.PasswordHash, []byte("Example123!")) != nil {
		t.Fatal("invalid password hash")
	}
	var count int64
	tx.Model(&model.DepartmentMember{}).Where("user_id=? AND department_id=? AND role='MEMBER'", user.ID, dep.ID).Count(&count)
	if count != 1 {
		t.Fatal("membership missing")
	}
	loginResponse := request("/api/v1/login", `{"username":"MARIA","password":"Example123!"}`, 200)
	var loginResult struct{ User directoryUser }
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginResult); err != nil {
		t.Fatal(err)
	}
	if len(loginResult.User.Departments) != 1 || loginResult.User.Departments[0].ID != dep.ID || loginResult.User.Departments[0].Role != "MEMBER" {
		t.Fatalf("missing login department: %s", loginResponse.Body.String())
	}
	request("/api/v1/login", `{"username":"maria","password":"wrong"}`, 401)
	request("/api/v1/login", `{"username":"missing","password":"wrong"}`, 401)
	request("/api/v1/users", body, 409)
	bad := strings.ReplaceAll(body, strconv.FormatInt(dep.ID, 10), "99999999")
	request("/api/v1/users", bad, 400)
	for _, path := range []string{"/api/v1/users?page=1&page_size=1&department_id=" + strconv.FormatInt(dep.ID, 10), "/api/v1/departments?page=1&page_size=1&q=SIS"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("list: %d %s", w.Code, w.Body.String())
		}
		var result struct {
			Data       []json.RawMessage
			Total      int
			TotalPages int `json:"total_pages"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || result.TotalPages != 1 || len(result.Data) != 1 {
			t.Fatalf("invalid page: %s", w.Body.String())
		}
		if strings.Contains(w.Body.String(), "password") {
			t.Fatal("credentials exposed")
		}
		if strings.Contains(path, "/users") {
			var row directoryUser
			if err := json.Unmarshal(result.Data[0], &row); err != nil {
				t.Fatal(err)
			}
			if len(row.Departments) != 1 || row.Departments[0].ID != dep.ID {
				t.Fatalf("areas missing: %s", w.Body.String())
			}
		}
	}
	for _, path := range []string{"/api/v1/users?department_id=invalid", "/api/v1/users?page_size=101", "/api/v1/departments?page=0", "/api/v1/users?admin=invalid"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/users?department_id="+"99999999", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("empty filter: %s", w.Body.String())
	}
	child := model.Department{Name: "Soporte", Code: "SUP", ParentID: &dep.ID, IsActive: true}
	if err := tx.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	var dt model.DocumentType
	if err := tx.Where("code='CARTA'").First(&dt).Error; err != nil {
		t.Fatal(err)
	}
	first, err := reserveDocumentNumber(tx, dt.ID, &dep.ID, 2026, dt.Code)
	if err != nil {
		t.Fatal(err)
	}
	second, err := reserveDocumentNumber(tx, dt.ID, &dep.ID, 2026, dt.Code)
	if err != nil {
		t.Fatal(err)
	}
	other, err := reserveDocumentNumber(tx, dt.ID, &child.ID, 2026, dt.Code)
	if err != nil {
		t.Fatal(err)
	}
	if first != "STD-1-CARTA" || second != "STD-2-CARTA" || other != "STD-1-CARTA" {
		t.Fatal("invalid independent sequence")
	}
	tx.SavePoint("sequence_reservation")
	rolled, err := reserveDocumentNumber(tx, dt.ID, &dep.ID, 2026, dt.Code)
	if err != nil {
		t.Fatal(err)
	}
	tx.RollbackTo("sequence_reservation")
	retry, err := reserveDocumentNumber(tx, dt.ID, &dep.ID, 2026, dt.Code)
	if err != nil || retry != rolled {
		t.Fatal("reservation rollback failed")
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/departments?q=SUP&page_size=1", nil))
	var areaPage struct {
		Data  []departmentWithParent
		Total int
	}
	if err := json.Unmarshal(w.Body.Bytes(), &areaPage); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || areaPage.Total != 1 || len(areaPage.Data) != 1 || areaPage.Data[0].Parent == nil || areaPage.Data[0].Parent.ID != dep.ID {
		t.Fatalf("parent missing: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/departments/organization-chart", nil))
	var chart OrganizationChart
	if err := json.Unmarshal(w.Body.Bytes(), &chart); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || chart.Total != 2 || chart.RootCount != 1 || len(chart.Roots[0].Children) != 1 || chart.Roots[0].Children[0].ID != child.ID {
		t.Fatalf("invalid chart: %s", w.Body.String())
	}
	docBody := `{"document_type_id":` + strconv.FormatInt(dt.ID, 10) + `,"department_id":` + strconv.FormatInt(dep.ID, 10) + `,"created_by":` + strconv.FormatInt(user.ID, 10) + `,"name":"Carta de prueba","file_url":"https://example.com/carta.pdf"}`
	var firstDoc, secondDoc model.Document
	w = request("/api/v1/documents", docBody, 201)
	if err := json.Unmarshal(w.Body.Bytes(), &firstDoc); err != nil {
		t.Fatal(err)
	}
	w = request("/api/v1/documents", docBody, 201)
	if err := json.Unmarshal(w.Body.Bytes(), &secondDoc); err != nil {
		t.Fatal(err)
	}
	if firstDoc.DocumentNumber == nil || secondDoc.DocumentNumber == nil || firstDoc.DocumentSequenceID == nil || secondDoc.DocumentSequenceID == nil || *firstDoc.DocumentSequenceID != *secondDoc.DocumentSequenceID || firstDoc.CorrelativeNumber == nil || secondDoc.CorrelativeNumber == nil || *secondDoc.CorrelativeNumber != *firstDoc.CorrelativeNumber+1 || !strings.HasPrefix(*firstDoc.DocumentNumber, "STD-") || !strings.HasSuffix(*firstDoc.DocumentNumber, "-SIS") {
		t.Fatal("invalid document sequence reference")
	}
	request("/api/v1/documents", strings.ReplaceAll(docBody, "https://example.com/carta.pdf", "invalid"), 400)
	request("/api/v1/documents", strings.ReplaceAll(docBody, `"department_id":`+strconv.FormatInt(dep.ID, 10), `"department_id":`+strconv.FormatInt(child.ID, 10)), 403)
	var sequence model.DocumentSequence
	if err := tx.First(&sequence, "id=?", *secondDoc.DocumentSequenceID).Error; err != nil {
		t.Fatal(err)
	}
	if sequence.LastNumber != *secondDoc.CorrelativeNumber {
		t.Fatal("counter mismatch")
	}
	noSeq := model.DocumentType{Name: "Adjunto test", Code: "ADJ_TEST", HasCorrelative: false, IsActive: true}
	if err := tx.Create(&noSeq).Error; err != nil {
		t.Fatal(err)
	}
	w = request("/api/v1/documents", strings.ReplaceAll(docBody, `"document_type_id":`+strconv.FormatInt(dt.ID, 10), `"document_type_id":`+strconv.FormatInt(noSeq.ID, 10)), 201)
	var attachment model.Document
	if err := json.Unmarshal(w.Body.Bytes(), &attachment); err != nil {
		t.Fatal(err)
	}
	if attachment.DocumentNumber != nil || attachment.DocumentSequenceID != nil || attachment.CorrelativeNumber != nil {
		t.Fatal("unexpected correlative")
	}
	for _, path := range []string{
		"/api/v1/documents?page_size=1&document_type_id=" + strconv.FormatInt(dt.ID, 10) + "&department_id=" + strconv.FormatInt(dep.ID, 10) + "&sort_by=correlative_number&order=asc",
		"/api/v1/documents/" + strconv.FormatInt(firstDoc.ID, 10),
		"/api/v1/document-types?page_size=1&q=CARTA&has_correlative=true&sort_by=code&order=desc",
	} {
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if strings.Contains(path, "page_size=1&document_type_id") {
			var page struct {
				Data  []documentView
				Total int
			}
			if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			if page.Total != 2 || len(page.Data) != 1 || page.Data[0].ID != firstDoc.ID || page.Data[0].DocumentType.ID != dt.ID || page.Data[0].DocumentType.Name != dt.Name || page.Data[0].Department == nil || page.Data[0].Department.ID != dep.ID || page.Data[0].Year == nil {
				t.Fatalf("invalid documents: %s", w.Body.String())
			}
		}
	}
	for _, path := range []string{"/api/v1/documents?sort_by=invalid", "/api/v1/documents?order=invalid", "/api/v1/documents?year=no", "/api/v1/documents?department_id=no", "/api/v1/document-types?is_active=no", "/api/v1/document-types?page_size=101"} {
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/documents/"+"99999999", nil))
	if w.Code != 404 {
		t.Fatal("missing document should return 404")
	}
	tx.Model(&model.User{}).Where("id=?", user.ID).Update("is_active", false)
	request("/api/v1/login", `{"username":"maria","password":"Example123!"}`, 401)
}
