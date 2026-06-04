package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ── ListMembers ──────────────────────────────────────────────────────────────

func (a *API) ListMembers(w http.ResponseWriter, r *http.Request, params openapi.ListMembersParams) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	page := 1
	pageSize := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PageSize != nil && *params.PageSize > 0 {
		pageSize = *params.PageSize
	}
	offset := (page - 1) * pageSize

	where := "1=1"
	args := []interface{}{}
	if params.Search != nil && *params.Search != "" {
		like := "%" + *params.Search + "%"
		where = `("firstName" LIKE ? OR "lastName" LIKE ? OR "email" LIKE ? OR CAST("number" AS TEXT) LIKE ?)`
		args = append(args, like, like, like, like)
	}

	var total int
	if err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM "Member" WHERE %s`, where), args...).Scan(&total); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to count members"})
		return
	}

	query := fmt.Sprintf(`SELECT "id","number","firstName","lastName","email","birthday","phone","phoneMobile","comment","entryDate","exitDate","street","postalCode","city","state","accountHolder","iban","bic","bankName","sepaMandateDate" FROM "Member" WHERE %s ORDER BY "lastName","firstName" LIMIT ? OFFSET ?`, where)
	args = append(args, pageSize, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to query members"})
		return
	}
	defer rows.Close()

	members := make([]openapi.Member, 0)
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan member"})
			return
		}
		members = append(members, m)
	}

	// Populate relations for each member
	for i := range members {
		members[i].Roles = fetchMemberNamedEntities(db, members[i].Id, "_MemberRoles", "Role")
		members[i].Groups = fetchMemberNamedEntities(db, members[i].Id, "_MemberGroups", "Group")
		members[i].Sections = fetchMemberNamedEntities(db, members[i].Id, "_MemberSections", "ClubSection")
	}

	totalPages := (total + pageSize - 1) / pageSize
	writeJSON(w, http.StatusOK, openapi.MembersListResponse{
		Items:      members,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// ── CreateMember ─────────────────────────────────────────────────────────────

func (a *API) CreateMember(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	var req openapi.MemberCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	res, err := db.Exec(
		`INSERT INTO "Member" ("number","firstName","lastName","email","birthday","phone","phoneMobile","comment","entryDate","exitDate","street","postalCode","city","state","accountHolder","iban","bic","bankName","sepaMandateDate") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.Number, req.FirstName, req.LastName, string(req.Email),
		optTime(req.Birthday), optStr(req.Phone), optStr(req.PhoneMobile), optStr(req.Comment),
		optTime(req.EntryDate), optTime(req.ExitDate), optStr(req.Street), optStr(req.PostalCode),
		optStr(req.City), optStr(req.State), optStr(req.AccountHolder), optStr(req.Iban),
		optStr(req.Bic), optStr(req.BankName), optTime(req.SepaMandateDate),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "member number or email already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create member"})
		return
	}

	id64, _ := res.LastInsertId()
	memberID := int(id64)

	syncMemberRelations(db, memberID, req.RoleIds, req.Roles, "_MemberRoles")
	syncMemberRelations(db, memberID, req.GroupIds, req.Groups, "_MemberGroups")
	syncMemberRelations(db, memberID, req.SectionIds, req.Sections, "_MemberSections")

	dbpkg.LogAudit(db, "CREATE", "Member", memberID, currentUserID(r), "")

	member, _ := getMemberByID(db, memberID)
	writeJSON(w, http.StatusCreated, member)
}

// ── GetMemberById ─────────────────────────────────────────────────────────────

func (a *API) GetMemberById(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	member, err := getMemberByID(a.getDB(), id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "member not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to fetch member"})
		return
	}
	writeJSON(w, http.StatusOK, member)
}

// ── UpdateMember ─────────────────────────────────────────────────────────────

func (a *API) UpdateMember(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	var req openapi.MemberUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	_, err := db.Exec(
		`UPDATE "Member" SET "number"=?,"firstName"=?,"lastName"=?,"email"=?,"birthday"=?,"phone"=?,"phoneMobile"=?,"comment"=?,"entryDate"=?,"exitDate"=?,"street"=?,"postalCode"=?,"city"=?,"state"=?,"accountHolder"=?,"iban"=?,"bic"=?,"bankName"=?,"sepaMandateDate"=? WHERE "id"=?`,
		req.Number, req.FirstName, req.LastName, string(req.Email),
		optTime(req.Birthday), optStr(req.Phone), optStr(req.PhoneMobile), optStr(req.Comment),
		optTime(req.EntryDate), optTime(req.ExitDate), optStr(req.Street), optStr(req.PostalCode),
		optStr(req.City), optStr(req.State), optStr(req.AccountHolder), optStr(req.Iban),
		optStr(req.Bic), optStr(req.BankName), optTime(req.SepaMandateDate), id,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "member number or email already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update member"})
		return
	}

	syncMemberRelations(db, id, req.RoleIds, req.Roles, "_MemberRoles")
	syncMemberRelations(db, id, req.GroupIds, req.Groups, "_MemberGroups")
	syncMemberRelations(db, id, req.SectionIds, req.Sections, "_MemberSections")

	dbpkg.LogAudit(db, "UPDATE", "Member", id, currentUserID(r), "")

	member, err := getMemberByID(db, id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "member not found"})
		return
	}
	writeJSON(w, http.StatusOK, member)
}

// ── DeleteMember ─────────────────────────────────────────────────────────────

func (a *API) DeleteMember(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	res, err := db.Exec(`DELETE FROM "Member" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete member"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "member not found"})
		return
	}

	dbpkg.LogAudit(db, "DELETE", "Member", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ── internal helpers ──────────────────────────────────────────────────────────

func getMemberByID(db *sql.DB, id int) (openapi.Member, error) {
	row := db.QueryRow(
		`SELECT "id","number","firstName","lastName","email","birthday","phone","phoneMobile","comment","entryDate","exitDate","street","postalCode","city","state","accountHolder","iban","bic","bankName","sepaMandateDate" FROM "Member" WHERE "id"=?`, id,
	)
	m, err := scanMemberRow(row)
	if err != nil {
		return m, err
	}
	m.Roles = fetchMemberNamedEntities(db, id, "_MemberRoles", "Role")
	m.Groups = fetchMemberNamedEntities(db, id, "_MemberGroups", "Group")
	m.Sections = fetchMemberNamedEntities(db, id, "_MemberSections", "ClubSection")
	return m, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanMemberRow(row rowScanner) (openapi.Member, error) {
	var m openapi.Member
	var email string
	var birthday, phone, phoneMobile, comment, entryDate, exitDate sql.NullString
	var street, postalCode, city, state, accountHolder, iban, bic, bankName, sepaMandateDate sql.NullString

	err := row.Scan(
		&m.Id, &m.Number, &m.FirstName, &m.LastName, &email,
		&birthday, &phone, &phoneMobile, &comment,
		&entryDate, &exitDate, &street, &postalCode, &city, &state,
		&accountHolder, &iban, &bic, &bankName, &sepaMandateDate,
	)
	if err != nil {
		return m, err
	}
	m.Email = openapi_types.Email(email)
	m.Birthday = parseOptTime(birthday)
	m.Phone = nullStrPtr(phone)
	m.PhoneMobile = nullStrPtr(phoneMobile)
	m.Comment = nullStrPtr(comment)
	m.EntryDate = parseOptTime(entryDate)
	m.ExitDate = parseOptTime(exitDate)
	m.Street = nullStrPtr(street)
	m.PostalCode = nullStrPtr(postalCode)
	m.City = nullStrPtr(city)
	m.State = nullStrPtr(state)
	m.AccountHolder = nullStrPtr(accountHolder)
	m.Iban = nullStrPtr(iban)
	m.Bic = nullStrPtr(bic)
	m.BankName = nullStrPtr(bankName)
	m.SepaMandateDate = parseOptTime(sepaMandateDate)
	m.Roles = []openapi.NamedEntity{}
	m.Groups = []openapi.NamedEntity{}
	m.Sections = []openapi.NamedEntity{}
	return m, nil
}

func scanMember(rows *sql.Rows) (openapi.Member, error) {
	var m openapi.Member
	var email string
	var birthday, phone, phoneMobile, comment, entryDate, exitDate sql.NullString
	var street, postalCode, city, state, accountHolder, iban, bic, bankName, sepaMandateDate sql.NullString

	err := rows.Scan(
		&m.Id, &m.Number, &m.FirstName, &m.LastName, &email,
		&birthday, &phone, &phoneMobile, &comment,
		&entryDate, &exitDate, &street, &postalCode, &city, &state,
		&accountHolder, &iban, &bic, &bankName, &sepaMandateDate,
	)
	if err != nil {
		return m, err
	}
	m.Email = openapi_types.Email(email)
	m.Birthday = parseOptTime(birthday)
	m.Phone = nullStrPtr(phone)
	m.PhoneMobile = nullStrPtr(phoneMobile)
	m.Comment = nullStrPtr(comment)
	m.EntryDate = parseOptTime(entryDate)
	m.ExitDate = parseOptTime(exitDate)
	m.Street = nullStrPtr(street)
	m.PostalCode = nullStrPtr(postalCode)
	m.City = nullStrPtr(city)
	m.State = nullStrPtr(state)
	m.AccountHolder = nullStrPtr(accountHolder)
	m.Iban = nullStrPtr(iban)
	m.Bic = nullStrPtr(bic)
	m.BankName = nullStrPtr(bankName)
	m.SepaMandateDate = parseOptTime(sepaMandateDate)
	m.Roles = []openapi.NamedEntity{}
	m.Groups = []openapi.NamedEntity{}
	m.Sections = []openapi.NamedEntity{}
	return m, nil
}

// fetchMemberNamedEntities loads the role/group/section list for a member via
// the given junction table and entity table.
func fetchMemberNamedEntities(db *sql.DB, memberID int, junctionTable, entityTable string) []openapi.NamedEntity {
	query := fmt.Sprintf(
		`SELECT e."id", e."name" FROM "%s" e JOIN "%s" j ON e."id" = j."B" WHERE j."A" = ?`,
		entityTable, junctionTable,
	)
	// _MemberRoles: A=Member, B=Role
	// _MemberGroups: A=Group, B=Member  — note reversed!
	// Actually need to check the Prisma convention…
	// Prisma uses alphabetical order: _MemberGroups A=Group B=Member → need where B=memberID
	// But _MemberRoles A=Member B=Role → need where A=memberID
	// Let's use the correct column based on the junction table.
	if junctionTable == "_MemberGroups" || junctionTable == "_MemberSections" {
		query = fmt.Sprintf(
			`SELECT e."id", e."name" FROM "%s" e JOIN "%s" j ON e."id" = j."A" WHERE j."B" = ?`,
			entityTable, junctionTable,
		)
	}

	rows, err := db.Query(query, memberID)
	if err != nil {
		return []openapi.NamedEntity{}
	}
	defer rows.Close()

	entities := []openapi.NamedEntity{}
	for rows.Next() {
		var e openapi.NamedEntity
		if err := rows.Scan(&e.Id, &e.Name); err == nil {
			entities = append(entities, e)
		}
	}
	return entities
}

// syncMemberRelations replaces the junction-table entries for a member.
// It accepts either a list of IDs or NamedEntityReferences.
func syncMemberRelations(db *sql.DB, memberID int, ids *[]int, refs *[]openapi.NamedEntityReference, junctionTable string) {
	collected := collectIDs(ids, refs)
	if collected == nil {
		return // not provided in request – leave unchanged
	}

	colA, colB := "A", "B"
	if junctionTable == "_MemberGroups" || junctionTable == "_MemberSections" {
		// Prisma convention: A=entity(Group/Section), B=Member
		colA, colB = "B", "A"
	}

	_, _ = db.Exec(fmt.Sprintf(`DELETE FROM "%s" WHERE "%s"=?`, junctionTable, colA), memberID)
	for _, entityID := range collected {
		_, _ = db.Exec(
			fmt.Sprintf(`INSERT OR IGNORE INTO "%s" ("%s","%s") VALUES (?,?)`, junctionTable, colA, colB),
			memberID, entityID,
		)
	}
}

func collectIDs(ids *[]int, refs *[]openapi.NamedEntityReference) []int {
	if ids != nil {
		return *ids
	}
	if refs != nil {
		result := make([]int, 0, len(*refs))
		for _, r := range *refs {
			result = append(result, r.Id)
		}
		return result
	}
	return nil
}

// ── time / null helpers ───────────────────────────────────────────────────────

func optTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func optStr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func nullStrPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

var timeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseOptTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	for _, format := range timeFormats {
		if t, err := time.Parse(format, ns.String); err == nil {
			return &t
		}
	}
	return nil
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

