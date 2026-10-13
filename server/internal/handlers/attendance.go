package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/pdfdoc"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

const maxAttendanceRangeDays = 1100

type attendanceOccurrenceJSON struct {
	EventID        string                `json:"eventId"`
	OccurrenceDate string                `json:"occurrenceDate"`
	Title          string                `json:"title"`
	Category       string                `json:"category"`
	Start          time.Time             `json:"start"`
	End            time.Time             `json:"end"`
	AllDay         bool                  `json:"allDay"`
	Groups         []openapi.NamedEntity `json:"groups"`
	Sections       []openapi.NamedEntity `json:"sections"`
	Roles          []openapi.NamedEntity `json:"roles"`
	Present        int                   `json:"present"`
	Excused        int                   `json:"excused"`
	Unexcused      int                   `json:"unexcused"`
}

type attendanceMemberJSON struct {
	honorMemberJSON
	Status     *string `json:"status,omitempty"`
	InAudience bool    `json:"inAudience"`
}

type attendanceSheetJSON struct {
	Occurrence attendanceOccurrenceJSON `json:"occurrence"`
	Members    []attendanceMemberJSON   `json:"members"`
}

type attendanceSetRequest struct {
	EventID string `json:"eventId"`
	Date    string `json:"date"`
	Entries []struct {
		MemberID string  `json:"memberId"`
		Status   *string `json:"status"`
	} `json:"entries"`
}

type memberAttendanceJSON struct {
	Member    honorMemberJSON `json:"member"`
	Present   int             `json:"present"`
	Excused   int             `json:"excused"`
	Unexcused int             `json:"unexcused"`
	Rate      float64         `json:"rate"`
}

type attendanceStatsJSON struct {
	Members     []memberAttendanceJSON     `json:"members"`
	Occurrences []attendanceOccurrenceJSON `json:"occurrences"`
}

func toAttendanceOccurrence(c store.AttendanceCount) attendanceOccurrenceJSON {
	o, e := c.Occurrence, c.Occurrence.Event
	return attendanceOccurrenceJSON{
		EventID: e.ID, OccurrenceDate: o.Date, Title: e.Title, Category: e.Category, Start: o.Start, End: o.End,
		AllDay: e.AllDay, Groups: toAPINamedList(e.Groups), Sections: toAPINamedList(e.Sections), Roles: toAPINamedList(e.Roles),
		Present: c.Present, Excused: c.Excused, Unexcused: c.Unexcused,
	}
}

// attendanceAccess checks the login and returns the events it may see.
// Writing needs a board role or TRAINER on a device that may change member
// data; trainers are limited to their own groups and sections.
func (a *API) attendanceAccess(w http.ResponseWriter, r *http.Request, st *store.Store, write bool) (store.AttendanceScope, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok && a.collab.CanRead(r.Context(), db.ScopeMembersCore)
	if allowed && write {
		allowed = (canEditData(claims.AppRole) || claims.AppRole == openapi.TRAINER) && a.collab.CanWrite(r.Context(), db.ScopeMembersCore)
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access attendance"})
		return store.AttendanceScope{}, false
	}
	if claims.AppRole != openapi.TRAINER {
		return store.AttendanceScope{All: true}, true
	}
	sc, err := st.TrainerScope(r.Context(), claims.Email)
	if err != nil {
		writeStoreError(w, err, "", "failed to load trainer groups")
		return sc, false
	}
	return sc, true
}

func forbidEvent(w http.ResponseWriter) {
	code := "NOT_YOUR_GROUP"
	writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "this event is not for one of your groups"})
}

// parseRange reads from/to (YYYY-MM-DD, both required).
func parseRange(w http.ResponseWriter, r *http.Request, maxDays int) (time.Time, time.Time, bool) {
	from, err1 := time.Parse(dateLayout, r.URL.Query().Get("from"))
	to, err2 := time.Parse(dateLayout, r.URL.Query().Get("to"))
	if err1 != nil || err2 != nil || to.Before(from) || to.Sub(from) > time.Duration(maxDays)*24*time.Hour {
		badRequest(w, fmt.Sprintf("from and to (YYYY-MM-DD) are required, at most %d days apart", maxDays))
		return from, to, false
	}
	return from, to, true
}

func attendanceFilter(r *http.Request) store.AttendanceFilter {
	return store.AttendanceFilter{GroupID: r.URL.Query().Get("groupId"), SectionID: r.URL.Query().Get("sectionId")}
}

// ListAttendanceOccurrences lists the occurrences the login may record, with
// counts: GET /api/v2/attendance/occurrences?from=&to=&groupId=&sectionId=.
func (a *API) ListAttendanceOccurrences(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	sc, ok := a.attendanceAccess(w, r, st, false)
	if !ok {
		return
	}
	from, to, ok := parseRange(w, r, 400)
	if !ok {
		return
	}
	counts, err := st.AttendanceOccurrences(r.Context(), from, to, sc, attendanceFilter(r))
	if err != nil {
		writeStoreError(w, err, "", "failed to list occurrences")
		return
	}
	out := make([]attendanceOccurrenceJSON, 0, len(counts))
	for _, c := range counts {
		out = append(out, toAttendanceOccurrence(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) writeSheet(w http.ResponseWriter, r *http.Request, st *store.Store, eventID string, date time.Time) {
	sheet, err := st.AttendanceSheet(r.Context(), eventID, date)
	if err != nil {
		writeStoreError(w, err, "event has no occurrence on this date", "failed to load attendance")
		return
	}
	c := store.AttendanceCount{Occurrence: sheet.Occurrence}
	out := attendanceSheetJSON{Members: make([]attendanceMemberJSON, 0, len(sheet.Members))}
	for _, m := range sheet.Members {
		out.Members = append(out.Members, attendanceMemberJSON{honorMemberJSON: memberRefJSON(m.MemberRef), Status: m.Status, InAudience: m.InAudience})
		if m.Status != nil {
			switch *m.Status {
			case store.AttendancePresent:
				c.Present++
			case store.AttendanceExcused:
				c.Excused++
			case store.AttendanceUnexcused:
				c.Unexcused++
			}
		}
	}
	out.Occurrence = toAttendanceOccurrence(c)
	writeJSON(w, http.StatusOK, out)
}

// GetAttendanceSheet: GET /api/v2/attendance/sheet?eventId=&date=.
func (a *API) GetAttendanceSheet(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	sc, ok := a.attendanceAccess(w, r, st, false)
	if !ok {
		return
	}
	eventID := r.URL.Query().Get("eventId")
	date, err := time.Parse(dateLayout, r.URL.Query().Get("date"))
	if eventID == "" || err != nil {
		badRequest(w, "eventId and date (YYYY-MM-DD) are required")
		return
	}
	e, err := st.GetEvent(r.Context(), eventID)
	if err != nil {
		writeStoreError(w, err, "event not found", "failed to load event")
		return
	}
	if !sc.Allows(&e) {
		forbidEvent(w)
		return
	}
	a.writeSheet(w, r, st, eventID, date)
}

// SetAttendance records several members at once and returns the sheet:
// PUT /api/v2/attendance/sheet. A null status clears an entry. Requests are
// idempotent, so clients may resend them after working offline.
func (a *API) SetAttendance(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	sc, ok := a.attendanceAccess(w, r, st, true)
	if !ok {
		return
	}
	var req attendanceSetRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	date, err := time.Parse(dateLayout, req.Date)
	if req.EventID == "" || err != nil || len(req.Entries) == 0 || len(req.Entries) > 1000 {
		badRequest(w, "eventId, date (YYYY-MM-DD) and 1 to 1000 entries are required")
		return
	}
	entries := make([]store.AttendanceEntry, 0, len(req.Entries))
	for _, en := range req.Entries {
		if en.MemberID == "" || (en.Status != nil && !store.ValidAttendanceStatus(*en.Status)) {
			badRequest(w, "each entry needs a memberId and a status of present, excused, unexcused or null")
			return
		}
		entries = append(entries, store.AttendanceEntry{MemberID: en.MemberID, Status: en.Status})
	}
	e, err := st.GetEvent(r.Context(), req.EventID)
	if err != nil {
		writeStoreError(w, err, "event not found", "failed to load event")
		return
	}
	if !sc.Allows(&e) {
		forbidEvent(w)
		return
	}
	if err := st.SetAttendance(r.Context(), req.EventID, date, entries); err != nil {
		writeStoreError(w, err, "event has no occurrence on this date", "failed to save attendance")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Attendance", req.EventID+"/"+req.Date, currentUserID(r))
	a.writeSheet(w, r, st, req.EventID, date)
}

// AttendanceStats returns the attendance per member and per occurrence:
// GET /api/v2/attendance/stats?from=&to=&groupId=&sectionId=.
func (a *API) AttendanceStats(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	sc, ok := a.attendanceAccess(w, r, st, false)
	if !ok {
		return
	}
	from, to, ok := parseRange(w, r, maxAttendanceRangeDays)
	if !ok {
		return
	}
	f := attendanceFilter(r)
	records, err := st.AttendanceRecords(r.Context(), from, to, sc, f)
	if err != nil {
		writeStoreError(w, err, "", "failed to load attendance")
		return
	}
	counts, err := st.AttendanceOccurrences(r.Context(), from, to, sc, f)
	if err != nil {
		writeStoreError(w, err, "", "failed to load attendance")
		return
	}
	out := attendanceStatsJSON{Members: []memberAttendanceJSON{}, Occurrences: []attendanceOccurrenceJSON{}}
	for _, m := range store.SummarizeByMember(records) {
		out.Members = append(out.Members, memberAttendanceJSON{Member: memberRefJSON(m.Member), Present: m.Present,
			Excused: m.Excused, Unexcused: m.Unexcused, Rate: m.Rate()})
	}
	for _, c := range counts {
		// Only occurrences with recorded attendance make up the history.
		if c.Present+c.Excused+c.Unexcused > 0 {
			out.Occurrences = append(out.Occurrences, toAttendanceOccurrence(c))
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// attendanceLabels are the texts of the PDF export, by language.
var attendanceLabels = map[string]map[string]string{
	"de": {"title": "Anwesenheitsliste", "period": "Zeitraum", "filter": "Gruppe/Abteilung", "all": "alle Termine",
		"summary": "Übersicht je Mitglied", "name": "Name", "number": "Nr.", "present": "anwesend", "excused": "entschuldigt",
		"unexcused": "unentschuldigt", "rate": "Quote", "sessions": "Termine", "none": "Keine Anwesenheit erfasst.",
		"created": "Erstellt", "date": "02.01.2006"},
	"en": {"title": "Attendance list", "period": "Period", "filter": "Group/section", "all": "all events",
		"summary": "Summary per member", "name": "Name", "number": "No.", "present": "present", "excused": "excused",
		"unexcused": "unexcused", "rate": "Rate", "sessions": "Sessions", "none": "No attendance recorded.",
		"created": "Created", "date": "2006-01-02"},
	"fr": {"title": "Liste de présence", "period": "Période", "filter": "Groupe/section", "all": "tous les événements",
		"summary": "Récapitulatif par membre", "name": "Nom", "number": "N°", "present": "présent", "excused": "excusé",
		"unexcused": "non excusé", "rate": "Taux", "sessions": "Séances", "none": "Aucune présence enregistrée.",
		"created": "Créé le", "date": "02/01/2006"},
}

func requestLanguage(r *http.Request) string {
	if al := strings.ToLower(r.Header.Get("Accept-Language")); len(al) >= 2 {
		if _, ok := attendanceLabels[al[:2]]; ok {
			return al[:2]
		}
	}
	return "en"
}

// ExportAttendance downloads the attendance in a period as CSV or PDF:
// GET /api/v2/attendance/export?from=&to=&groupId=&sectionId=&format=csv|pdf.
func (a *API) ExportAttendance(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	sc, ok := a.attendanceAccess(w, r, st, false)
	if !ok {
		return
	}
	from, to, ok := parseRange(w, r, maxAttendanceRangeDays)
	if !ok {
		return
	}
	format := r.URL.Query().Get("format")
	if format != "csv" && format != "pdf" {
		badRequest(w, "format must be csv or pdf")
		return
	}
	f := attendanceFilter(r)
	records, err := st.AttendanceRecords(r.Context(), from, to, sc, f)
	if err != nil {
		writeStoreError(w, err, "", "failed to load attendance")
		return
	}
	name := fmt.Sprintf("attendance-%s-%s", from.Format(dateLayout), to.Format(dateLayout))

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"date", "start", "event", "number", "firstName", "lastName", "status"})
		for _, rec := range records {
			o := rec.Occurrence
			start := ""
			if !o.Event.AllDay {
				start = o.Start.Format("15:04")
			}
			_ = cw.Write([]string{o.Date, start, csvSafe(o.Event.Title), strconv.Itoa(rec.Member.Number),
				csvSafe(rec.Member.FirstName), csvSafe(rec.Member.LastName), rec.Status})
		}
		cw.Flush()
		return
	}

	l := attendanceLabels[requestLanguage(r)]
	filterName := l["all"]
	if f.GroupID != "" || f.SectionID != "" {
		kind, id := store.Groups, f.GroupID
		if f.GroupID == "" {
			kind, id = store.Sections, f.SectionID
		}
		if list, err := st.ListNamed(r.Context(), kind); err == nil {
			for _, n := range list {
				if n.ID == id {
					filterName = n.Name
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.pdf"`)
	_, _ = w.Write(attendancePDF(l, from, to, filterName, records))
}

// attendancePDF renders the summary per member and the list per occurrence.
func attendancePDF(l map[string]string, from, to time.Time, filterName string, records []store.AttendanceRecord) []byte {
	d := pdfdoc.New(false)
	date := func(t time.Time) string { return t.Format(l["date"]) }
	d.Line(16, true, l["title"])
	d.Line(10, false, l["period"]+": "+date(from)+" – "+date(to))
	d.Line(10, false, l["filter"]+": "+filterName)
	d.Line(10, false, l["created"]+": "+date(time.Now()))
	d.Row(10, false)
	if len(records) == 0 {
		d.Line(10, false, l["none"])
		return d.Bytes()
	}

	cols := []int{0, 40, 260, 320, 400, 485}
	d.Line(13, true, l["summary"])
	d.Row(9, true, pdfdoc.Cell{X: cols[0], Text: l["number"]}, pdfdoc.Cell{X: cols[1], Text: l["name"]},
		pdfdoc.Cell{X: cols[2], Text: l["present"]}, pdfdoc.Cell{X: cols[3], Text: l["excused"]},
		pdfdoc.Cell{X: cols[4], Text: l["unexcused"]}, pdfdoc.Cell{X: cols[5], Text: l["rate"]})
	for _, m := range store.SummarizeByMember(records) {
		d.Row(9, false, pdfdoc.Cell{X: cols[0], Text: strconv.Itoa(m.Member.Number)},
			pdfdoc.Cell{X: cols[1], Text: pdfdoc.Fit(m.Member.LastName+", "+m.Member.FirstName, 40)},
			pdfdoc.Cell{X: cols[2], Text: strconv.Itoa(m.Present)}, pdfdoc.Cell{X: cols[3], Text: strconv.Itoa(m.Excused)},
			pdfdoc.Cell{X: cols[4], Text: strconv.Itoa(m.Unexcused)},
			pdfdoc.Cell{X: cols[5], Text: fmt.Sprintf("%.0f %%", m.Rate()*100)})
	}

	d.Row(10, false)
	d.Line(13, true, l["sessions"])
	for i := 0; i < len(records); {
		o := records[i].Occurrence
		j := i
		present := 0
		for j < len(records) && records[j].Occurrence.Event.ID == o.Event.ID && records[j].Occurrence.Date == o.Date {
			if records[j].Status == store.AttendancePresent {
				present++
			}
			j++
		}
		heading := date(o.Start) + " " + o.Event.Title
		if !o.Event.AllDay {
			heading = date(o.Start) + " " + o.Start.Format("15:04") + " " + o.Event.Title
		}
		d.NeedSpace(3, 11)
		d.Row(4, false)
		d.Row(11, true, pdfdoc.Cell{Text: pdfdoc.Fit(heading, 70)}, pdfdoc.Cell{X: cols[4], Text: fmt.Sprintf("%d %s", present, l["present"])})
		for _, rec := range records[i:j] {
			d.Row(9, false, pdfdoc.Cell{X: cols[0], Text: strconv.Itoa(rec.Member.Number)},
				pdfdoc.Cell{X: cols[1], Text: pdfdoc.Fit(rec.Member.LastName+", "+rec.Member.FirstName, 40)},
				pdfdoc.Cell{X: cols[2], Text: l[rec.Status]})
		}
		i = j
	}
	return d.Bytes()
}
