package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

func (a *API) ListEvents(w http.ResponseWriter, _ *http.Request, params openapi.ListEventsParams) {
	if !a.requireDB(w) {
		return
	}

	query := `SELECT "id","title","description","location","startDate","endDate","type","maxParticipants","createdAt","updatedAt" FROM "Event"`
	filters := make([]string, 0, 3)
	args := make([]interface{}, 0, 3)

	if params.StartDateFrom != nil {
		filters = append(filters, `"startDate" >= ?`)
		args = append(args, params.StartDateFrom.UTC().Format(time.RFC3339))
	}
	if params.StartDateTo != nil {
		filters = append(filters, `"startDate" <= ?`)
		args = append(args, params.StartDateTo.UTC().Format(time.RFC3339))
	}
	if params.Type != nil {
		if !params.Type.Valid() {
			writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid event type"})
			return
		}
		filters = append(filters, `"type" = ?`)
		args = append(args, string(*params.Type))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += ` ORDER BY "startDate" ASC`

	rows, err := a.getDB().Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list events"})
		return
	}
	defer rows.Close()

	result := []openapi.Event{}
	for rows.Next() {
		e, scanErr := scanEvent(rows)
		if scanErr != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan event"})
			return
		}
		result = append(result, e)
	}

	writeJSON(w, http.StatusOK, result)
}

func (a *API) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}

	var req openapi.EventCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	if !req.Type.Valid() {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid event type"})
		return
	}
	if req.EndDate.Before(req.StartDate) {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "endDate must be after startDate"})
		return
	}
	if req.MaxParticipants != nil && *req.MaxParticipants <= 0 {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "maxParticipants must be greater than 0"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.getDB().Exec(
		`INSERT INTO "Event" ("title","description","location","startDate","endDate","type","maxParticipants","createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?,?)`,
		req.Title, optStr(req.Description), optStr(req.Location), req.StartDate.UTC().Format(time.RFC3339), req.EndDate.UTC().Format(time.RFC3339), string(req.Type), req.MaxParticipants, now, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create event"})
		return
	}

	id64, _ := res.LastInsertId()
	e, err := getEventByID(a.getDB(), int(id64))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load created event"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "CREATE", "Event", int(id64), currentUserID(r), "")
	writeJSON(w, http.StatusCreated, e)
}

func (a *API) GetEventById(w http.ResponseWriter, _ *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}

	e, err := getEventByID(a.getDB(), id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "event not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load event"})
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (a *API) UpdateEvent(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}

	var req openapi.EventUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := getEventByID(a.getDB(), id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "event not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load event"})
		return
	}

	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}
	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}
	location := existing.Location
	if req.Location != nil {
		location = req.Location
	}
	startDate := existing.StartDate
	if req.StartDate != nil {
		startDate = *req.StartDate
	}
	endDate := existing.EndDate
	if req.EndDate != nil {
		endDate = *req.EndDate
	}
	eventType := existing.Type
	if req.Type != nil {
		if !req.Type.Valid() {
			writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid event type"})
			return
		}
		eventType = *req.Type
	}
	maxParticipants := existing.MaxParticipants
	if req.MaxParticipants != nil {
		maxParticipants = req.MaxParticipants
	}
	if maxParticipants != nil && *maxParticipants <= 0 {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "maxParticipants must be greater than 0"})
		return
	}
	if endDate.Before(startDate) {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "endDate must be after startDate"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = a.getDB().Exec(
		`UPDATE "Event" SET "title"=?,"description"=?,"location"=?,"startDate"=?,"endDate"=?,"type"=?,"maxParticipants"=?,"updatedAt"=? WHERE "id"=?`,
		title, optStr(description), optStr(location), startDate.UTC().Format(time.RFC3339), endDate.UTC().Format(time.RFC3339), string(eventType), maxParticipants, now, id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update event"})
		return
	}

	updated, err := getEventByID(a.getDB(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load updated event"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "UPDATE", "Event", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, updated)
}

func (a *API) DeleteEvent(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}

	res, err := a.getDB().Exec(`DELETE FROM "Event" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete event"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "event not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", "Event", id, currentUserID(r), "")
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListEventAttendees(w http.ResponseWriter, _ *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}

	if !eventExists(a.getDB(), id) {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "event not found"})
		return
	}

	rows, err := a.getDB().Query(
		`SELECT ea."id",ea."eventId",ea."memberId",ea."status",ea."createdAt",m."id",m."firstName",m."lastName",m."number"
		FROM "EventAttendee" ea
		JOIN "Member" m ON m."id"=ea."memberId"
		WHERE ea."eventId"=?
		ORDER BY m."lastName", m."firstName"`,
		id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list attendees"})
		return
	}
	defer rows.Close()

	result := []openapi.EventAttendee{}
	for rows.Next() {
		var attendee openapi.EventAttendee
		var createdAtStr string
		var member openapi.MemberSummary

		if scanErr := rows.Scan(&attendee.Id, &attendee.EventId, &attendee.MemberId, (*string)(&attendee.Status), &createdAtStr, &member.Id, &member.FirstName, &member.LastName, &member.Number); scanErr != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan attendee"})
			return
		}
		createdAt, _ := parseTime(createdAtStr)
		attendee.CreatedAt = createdAt
		attendee.Member = &member
		result = append(result, attendee)
	}

	writeJSON(w, http.StatusOK, result)
}

func (a *API) UpsertEventAttendees(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}

	if !eventExists(a.getDB(), id) {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "event not found"})
		return
	}

	var req openapi.EventAttendeeBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	tx, err := a.getDB().Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update attendees"})
		return
	}
	defer tx.Rollback()

	for _, item := range req.Attendees {
		status := openapi.Registered
		if item.Status != nil {
			if !item.Status.Valid() {
				writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid attendee status"})
				return
			}
			status = *item.Status
		}

		res, execErr := tx.Exec(`UPDATE "EventAttendee" SET "status"=? WHERE "eventId"=? AND "memberId"=?`, string(status), id, item.MemberId)
		if execErr != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update attendee"})
			return
		}

		if rowsUpdated, _ := res.RowsAffected(); rowsUpdated == 0 {
			_, execErr = tx.Exec(
				`INSERT INTO "EventAttendee" ("eventId","memberId","status","createdAt") VALUES (?,?,?,?)`,
				id, item.MemberId, string(status), time.Now().UTC().Format(time.RFC3339),
			)
			if execErr != nil {
				writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to register attendee"})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update attendees"})
		return
	}

	dbpkg.LogAudit(a.getDB(), "UPDATE", "EventAttendee", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

func (a *API) DeleteEventAttendee(w http.ResponseWriter, r *http.Request, id int, memberId int) {
	if !a.requireDB(w) {
		return
	}

	res, err := a.getDB().Exec(`DELETE FROM "EventAttendee" WHERE "eventId"=? AND "memberId"=?`, id, memberId)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to remove attendee"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "event attendee not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", "EventAttendee", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

func eventExists(db *sql.DB, id int) bool {
	var exists int
	if err := db.QueryRow(`SELECT 1 FROM "Event" WHERE "id"=?`, id).Scan(&exists); err != nil {
		return false
	}
	return true
}

func getEventByID(db *sql.DB, id int) (openapi.Event, error) {
	row := db.QueryRow(`SELECT "id","title","description","location","startDate","endDate","type","maxParticipants","createdAt","updatedAt" FROM "Event" WHERE "id"=?`, id)
	return scanEventRow(row)
}

type eventRows interface {
	Scan(dest ...interface{}) error
}

func scanEvent(rows *sql.Rows) (openapi.Event, error) {
	var event openapi.Event
	var description, location sql.NullString
	var maxParticipants sql.NullInt64
	var startDateStr, endDateStr, createdAtStr, updatedAtStr string

	err := rows.Scan(
		&event.Id,
		&event.Title,
		&description,
		&location,
		&startDateStr,
		&endDateStr,
		(*string)(&event.Type),
		&maxParticipants,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		return event, err
	}
	mapEventNullableFields(&event, description, location, maxParticipants, startDateStr, endDateStr, createdAtStr, updatedAtStr)
	return event, nil
}

func scanEventRow(row eventRows) (openapi.Event, error) {
	var event openapi.Event
	var description, location sql.NullString
	var maxParticipants sql.NullInt64
	var startDateStr, endDateStr, createdAtStr, updatedAtStr string

	err := row.Scan(
		&event.Id,
		&event.Title,
		&description,
		&location,
		&startDateStr,
		&endDateStr,
		(*string)(&event.Type),
		&maxParticipants,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		return event, err
	}
	mapEventNullableFields(&event, description, location, maxParticipants, startDateStr, endDateStr, createdAtStr, updatedAtStr)
	return event, nil
}

func mapEventNullableFields(event *openapi.Event, description sql.NullString, location sql.NullString, maxParticipants sql.NullInt64, startDateStr string, endDateStr string, createdAtStr string, updatedAtStr string) {
	event.Description = nullStrPtr(description)
	event.Location = nullStrPtr(location)
	if maxParticipants.Valid {
		value := int(maxParticipants.Int64)
		event.MaxParticipants = &value
	}
	event.StartDate, _ = parseTime(startDateStr)
	event.EndDate, _ = parseTime(endDateStr)
	event.CreatedAt, _ = parseTime(createdAtStr)
	event.UpdatedAt, _ = parseTime(updatedAtStr)
}
