package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

// ── ListMembers ──────────────────────────────────────────────────────────────

func (a *API) ListMembers(w http.ResponseWriter, r *http.Request, params openapi.ListMembersParams) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}

	page := 1
	pageSize := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PageSize != nil && *params.PageSize > 0 {
		pageSize = *params.PageSize
	}
	filter := store.MemberFilter{Limit: pageSize, Offset: (page - 1) * pageSize}
	if params.Search != nil {
		filter.Search = *params.Search
	}

	members, total, err := st.ListMembers(r.Context(), filter)
	if err != nil {
		writeStoreError(w, err, "", "failed to query members")
		return
	}

	items := make([]openapi.Member, len(members))
	for i, m := range members {
		items[i] = toAPIMember(m)
	}
	writeJSON(w, http.StatusOK, openapi.MembersListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

// ── CreateMember ─────────────────────────────────────────────────────────────

func (a *API) CreateMember(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.MemberCreateRequest
	if err := decodeMemberRequest(w, r, &req); err != nil {
		writeMemberDecodeError(w, err)
		return
	}

	in := memberInput(req.Number, req.FirstName, req.LastName, req.Email, req.Birthday, req.EntryDate, req.ExitDate,
		req.Comment, req.Phone, req.PhoneMobile, req.Street, req.PostalCode, req.City, req.State,
		req.AccountHolder, req.Iban, req.Bic, req.BankName, req.SepaMandateDate)
	in.RoleIDs = collectIDs(req.RoleIds, req.Roles)
	in.GroupIDs = collectIDs(req.GroupIds, req.Groups)
	in.SectionIDs = collectIDs(req.SectionIds, req.Sections)

	member, err := st.CreateMember(r.Context(), in)
	if err != nil {
		writeStoreError(w, err, "", "failed to create member")
		return
	}
	st.Audit(r.Context(), "CREATE", "Member", member.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toAPIMember(member))
}

// ── GetMemberById ─────────────────────────────────────────────────────────────

func (a *API) GetMemberById(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	member, err := st.GetMember(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to fetch member")
		return
	}
	writeJSON(w, http.StatusOK, toAPIMember(member))
}

// ── UpdateMember ─────────────────────────────────────────────────────────────

func (a *API) UpdateMember(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.MemberUpdateRequest
	if err := decodeMemberRequest(w, r, &req); err != nil {
		writeMemberDecodeError(w, err)
		return
	}

	in := memberInput(req.Number, req.FirstName, req.LastName, req.Email, req.Birthday, req.EntryDate, req.ExitDate,
		req.Comment, req.Phone, req.PhoneMobile, req.Street, req.PostalCode, req.City, req.State,
		req.AccountHolder, req.Iban, req.Bic, req.BankName, req.SepaMandateDate)
	in.RoleIDs = collectIDs(req.RoleIds, req.Roles)
	in.GroupIDs = collectIDs(req.GroupIds, req.Groups)
	in.SectionIDs = collectIDs(req.SectionIds, req.Sections)

	member, err := st.UpdateMember(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to update member")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Member", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPIMember(member))
}

// ── DeleteMember ─────────────────────────────────────────────────────────────

func (a *API) DeleteMember(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	if err := st.DeleteMember(r.Context(), id); err != nil {
		writeStoreError(w, err, "member not found", "failed to delete member")
		return
	}
	st.Audit(r.Context(), "DELETE", "Member", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ── request decoding ──────────────────────────────────────────────────────────

// maxMemberRequestBytes is far above what the field length limits allow.
const maxMemberRequestBytes = 1 << 16

func decodeMemberRequest(w http.ResponseWriter, r *http.Request, req any) error {
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, maxMemberRequestBytes)).Decode(req)
}

// writeMemberDecodeError answers a request body that could not be decoded.
// A malformed e-mail or a value of the wrong type is reported as a field
// problem like the checks of store.ValidateMemberInput.
func writeMemberDecodeError(w http.ResponseWriter, err error) {
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.Is(err, openapi_types.ErrValidationEmail):
		writeValidationError(w, []store.FieldError{{Field: "email", Code: store.CodeInvalid}})
	case errors.As(err, &typeErr) && typeErr.Field != "":
		writeValidationError(w, []store.FieldError{{Field: typeErr.Field, Code: store.CodeInvalid}})
	default:
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
	}
}

// ── mapping ───────────────────────────────────────────────────────────────────

// memberInput builds the store input from the (identical) fields of the create
// and update requests.
func memberInput(number int, firstName, lastName string, email openapi_types.Email,
	birthday, entryDate, exitDate *time.Time, comment *string,
	phone, phoneMobile, street, postalCode, city, state *string,
	accountHolder, iban, bic, bankName *string, sepaMandateDate *time.Time,
) store.MemberInput {
	var emailPtr *string
	if email != "" {
		e := string(email)
		emailPtr = &e
	}
	return store.MemberInput{
		Number: number, FirstName: firstName, LastName: lastName,
		Birthday: birthday, EntryDate: entryDate, ExitDate: exitDate, Comment: comment,
		Contact: store.MemberContact{
			Email: emailPtr, Phone: phone, PhoneMobile: phoneMobile,
			Street: street, PostalCode: postalCode, City: city, State: state,
		},
		Bank: store.MemberBank{
			AccountHolder: accountHolder, IBAN: iban, BIC: bic, BankName: bankName, SEPAMandateDate: sepaMandateDate,
		},
	}
}

func toAPIMember(m store.Member) openapi.Member {
	out := openapi.Member{
		Id: m.ID, Number: m.Number, FirstName: m.FirstName, LastName: m.LastName,
		Birthday: m.Birthday, EntryDate: m.EntryDate, ExitDate: m.ExitDate, Comment: m.Comment,
		Phone: m.Contact.Phone, PhoneMobile: m.Contact.PhoneMobile, Street: m.Contact.Street,
		PostalCode: m.Contact.PostalCode, City: m.Contact.City, State: m.Contact.State,
		AccountHolder: m.Bank.AccountHolder, Iban: m.Bank.IBAN, Bic: m.Bank.BIC, BankName: m.Bank.BankName,
		SepaMandateDate: m.Bank.SEPAMandateDate,
		Roles:           toAPINamedList(m.Roles),
		Groups:          toAPINamedList(m.Groups),
		Sections:        toAPINamedList(m.Sections),
	}
	if m.Contact.Email != nil && *m.Contact.Email != "" {
		email := openapi_types.Email(*m.Contact.Email)
		out.Email = &email
	}
	return out
}

func toAPINamedList(list []store.NamedEntity) []openapi.NamedEntity {
	out := make([]openapi.NamedEntity, len(list))
	for i, e := range list {
		out[i] = openapi.NamedEntity{Id: e.ID, Name: e.Name}
	}
	return out
}

func toAPIMemberSummary(m store.MemberSummary) openapi.MemberSummary {
	return openapi.MemberSummary{Id: m.ID, Number: m.Number, FirstName: m.FirstName, LastName: m.LastName}
}

// collectIDs returns the link ids of a request, given either as plain ids or as
// references; nil means "not provided".
func collectIDs(ids *[]string, refs *[]openapi.NamedEntityReference) *[]string {
	if ids != nil {
		return ids
	}
	if refs != nil {
		result := make([]string, 0, len(*refs))
		for _, r := range *refs {
			result = append(result, r.Id)
		}
		return &result
	}
	return nil
}
