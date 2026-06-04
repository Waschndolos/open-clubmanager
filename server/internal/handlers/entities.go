package handlers

import (
	"database/sql"
	"encoding/json"
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
