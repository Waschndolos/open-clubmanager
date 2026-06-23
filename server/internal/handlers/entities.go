package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ── Roles ────────────────────────────────────────────────────────────────────

func (a *API) ListRoles(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	entities, err := queryNamedEntities(a.getDB(), "Role")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list roles"})
		return
	}
	writeJSON(w, http.StatusOK, entities)
}

func (a *API) CreateRole(w http.ResponseWriter, r *http.Request) {
	a.createNamedEntity(w, r, "Role")
}

func (a *API) UpdateRole(w http.ResponseWriter, r *http.Request, id int) {
	a.updateNamedEntity(w, r, "Role", id)
}

func (a *API) DeleteRole(w http.ResponseWriter, r *http.Request, id int) {
	a.deleteNamedEntity(w, r, "Role", id)
}

func (a *API) GetRoleMembers(w http.ResponseWriter, r *http.Request, id int) {
	a.getEntityMembers(w, id, "_MemberRoles", "role")
}

func (a *API) BulkAssignRoleMembers(w http.ResponseWriter, r *http.Request, id int) {
	a.bulkAssignEntityMembers(w, r, id, "_MemberRoles", "role")
}

// ── Groups ───────────────────────────────────────────────────────────────────

func (a *API) ListGroups(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	entities, err := queryNamedEntities(a.getDB(), "Group")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list groups"})
		return
	}
	writeJSON(w, http.StatusOK, entities)
}

func (a *API) CreateGroup(w http.ResponseWriter, r *http.Request) {
	a.createNamedEntity(w, r, "Group")
}

func (a *API) UpdateGroup(w http.ResponseWriter, r *http.Request, id int) {
	a.updateNamedEntity(w, r, "Group", id)
}

func (a *API) DeleteGroup(w http.ResponseWriter, r *http.Request, id int) {
	a.deleteNamedEntity(w, r, "Group", id)
}

func (a *API) GetGroupMembers(w http.ResponseWriter, r *http.Request, id int) {
	a.getEntityMembers(w, id, "_MemberGroups", "group")
}

func (a *API) BulkAssignGroupMembers(w http.ResponseWriter, r *http.Request, id int) {
	a.bulkAssignEntityMembers(w, r, id, "_MemberGroups", "group")
}

// ── Sections ─────────────────────────────────────────────────────────────────

func (a *API) ListSections(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	entities, err := queryNamedEntities(a.getDB(), "ClubSection")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list sections"})
		return
	}
	writeJSON(w, http.StatusOK, entities)
}

func (a *API) CreateSection(w http.ResponseWriter, r *http.Request) {
	a.createNamedEntity(w, r, "ClubSection")
}

func (a *API) UpdateSection(w http.ResponseWriter, r *http.Request, id int) {
	a.updateNamedEntity(w, r, "ClubSection", id)
}

func (a *API) DeleteSection(w http.ResponseWriter, r *http.Request, id int) {
	a.deleteNamedEntity(w, r, "ClubSection", id)
}

func (a *API) GetSectionMembers(w http.ResponseWriter, r *http.Request, id int) {
	a.getEntityMembers(w, id, "_MemberSections", "section")
}

func (a *API) BulkAssignSectionMembers(w http.ResponseWriter, r *http.Request, id int) {
	a.bulkAssignEntityMembers(w, r, id, "_MemberSections", "section")
}

// ── generic named-entity helpers ──────────────────────────────────────────────

func queryNamedEntities(db *sql.DB, table string) ([]openapi.NamedEntity, error) {
	rows, err := db.Query(`SELECT "id","name" FROM "` + table + `" ORDER BY "name"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []openapi.NamedEntity{}
	for rows.Next() {
		var e openapi.NamedEntity
		if err := rows.Scan(&e.Id, &e.Name); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (a *API) createNamedEntity(w http.ResponseWriter, r *http.Request, table string) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.NamedEntityCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	res, err := a.getDB().Exec(`INSERT INTO "`+table+`" ("name") VALUES (?)`, req.Name)
	if err != nil {
		if isUniqueConstraint(err) {
			writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "name already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create entity"})
		return
	}
	id64, _ := res.LastInsertId()
	dbpkg.LogAudit(a.getDB(), "CREATE", table, int(id64), currentUserID(r), "")
	writeJSON(w, http.StatusCreated, openapi.NamedEntity{Id: int(id64), Name: req.Name})
}

func (a *API) updateNamedEntity(w http.ResponseWriter, r *http.Request, table string, id int) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.NamedEntityUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	res, err := a.getDB().Exec(`UPDATE "`+table+`" SET "name"=? WHERE "id"=?`, req.Name, id)
	if err != nil {
		if isUniqueConstraint(err) {
			writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "name already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update entity"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "entity not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "UPDATE", table, id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.NamedEntity{Id: id, Name: req.Name})
}

func (a *API) deleteNamedEntity(w http.ResponseWriter, r *http.Request, table string, id int) {
	if !a.requireDB(w) {
		return
	}
	res, err := a.getDB().Exec(`DELETE FROM "`+table+`" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete entity"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "entity not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", table, id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// getEntityMembers returns the list of members assigned to the given entity
// (group, role, or section) identified by junctionTable and entityID.
func (a *API) getEntityMembers(w http.ResponseWriter, entityID int, junctionTable string, entityLabel string) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	// Resolve the correct join column based on Prisma junction-table convention:
	// _MemberRoles:    A=Member, B=Role    → members are in column A where B=entityID
	// _MemberGroups:   A=Group,  B=Member  → members are in column B where A=entityID
	// _MemberSections: A=Section,B=Member  → members are in column B where A=entityID
	var query string
	if junctionTable == "_MemberRoles" {
		query = fmt.Sprintf(
			`SELECT m."id", m."firstName", m."lastName", m."number" FROM "Member" m JOIN "%s" j ON m."id" = j."A" WHERE j."B" = ? ORDER BY m."lastName", m."firstName"`,
			junctionTable,
		)
	} else {
		query = fmt.Sprintf(
			`SELECT m."id", m."firstName", m."lastName", m."number" FROM "Member" m JOIN "%s" j ON m."id" = j."B" WHERE j."A" = ? ORDER BY m."lastName", m."firstName"`,
			junctionTable,
		)
	}

	rows, err := db.Query(query, entityID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list " + entityLabel + " members"})
		return
	}
	defer rows.Close()

	members := []openapi.MemberSummary{}
	for rows.Next() {
		var m openapi.MemberSummary
		if err := rows.Scan(&m.Id, &m.FirstName, &m.LastName, &m.Number); err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan member"})
			return
		}
		members = append(members, m)
	}
	writeJSON(w, http.StatusOK, members)
}

// bulkAssignEntityMembers replaces all member assignments for the given entity.
// It accepts a BulkMemberAssignRequest and syncs the junction table accordingly.
func (a *API) bulkAssignEntityMembers(w http.ResponseWriter, r *http.Request, entityID int, junctionTable string, entityLabel string) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	var req openapi.BulkMemberAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	// Resolve correct column convention (same as syncMemberRelations):
	// _MemberRoles:    colA=Member, colB=Role    → entity is colB
	// _MemberGroups:   colA=Group,  colB=Member  → entity is colA
	// _MemberSections: colA=Section,colB=Member  → entity is colA
	entityCol, memberCol := "A", "B"
	if junctionTable == "_MemberRoles" {
		entityCol, memberCol = "B", "A"
	}

	_, err := db.Exec(fmt.Sprintf(`DELETE FROM "%s" WHERE "%s"=?`, junctionTable, entityCol), entityID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update " + entityLabel + " members"})
		return
	}

	for _, memberID := range req.MemberIds {
		_, err = db.Exec(
			fmt.Sprintf(`INSERT OR IGNORE INTO "%s" ("%s","%s") VALUES (?,?)`, junctionTable, entityCol, memberCol),
			entityID, memberID,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to assign member to " + entityLabel})
			return
		}
	}

	dbpkg.LogAudit(db, "UPDATE", junctionTable, entityID, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}
