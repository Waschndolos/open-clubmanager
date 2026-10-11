package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

// Databases created before the goose migrations used Prisma-style tables
// ("Member", "Role", ...) with integer ids. They are recognised by the
// "Member" table, backed up, and their data copied into the new schema with
// fresh UUIDs, after which the legacy tables are dropped.

// legacyTables in drop order (children before parents).
var legacyTables = []string{
	"_MemberRoles", "_MemberGroups", "_MemberSections", "_UserRoles",
	"InventoryLoan", "InventoryItem", "MemberFee", "FinanceTransaction",
	"Member", "Role", "Group", "ClubSection",
	"UserPreference", "RefreshToken", "PasswordResetToken", "User",
	"AuditLog", "_prisma_migrations",
}

func hasLegacySchema(ctx context.Context, db *sql.DB) (bool, error) {
	return tableExists(ctx, db, "Member")
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func tableExists(ctx context.Context, q queryer, name string) (bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?`, name)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}

// legacyRows reads all rows of a legacy query as column-name maps. Club
// databases are small enough for this to be fine. A missing table yields no
// rows.
func legacyRows(ctx context.Context, tx *sql.Tx, table, query string) ([]map[string]any, error) {
	ok, err := tableExists(ctx, tx, table)
	if err != nil || !ok {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", table, err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			m[c] = vals[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func legacyInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		return int64(x), true
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		return n, err == nil
	case []byte:
		return legacyInt(string(x))
	}
	return 0, false
}

func legacyFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	case []byte:
		return legacyFloat(string(x))
	}
	return 0, false
}

// legacyStr returns a nullable string for storage.
func legacyStr(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(x)
	}
}

// legacyTime returns a nullable normalized timestamp for storage.
func legacyTime(v any) any {
	if s, ok := normalizeLegacyTime(v); ok {
		return s
	}
	return nil
}

// legacyTimeOr returns a normalized timestamp, or fallback when absent.
func legacyTimeOr(v any, fallback string) string {
	if s, ok := normalizeLegacyTime(v); ok {
		return s
	}
	return fallback
}

type idMap map[int64]string

func (m idMap) assign(v any) (string, bool) {
	n, ok := legacyInt(v)
	if !ok {
		return "", false
	}
	id := NewID()
	m[n] = id
	return id, true
}

func (m idMap) lookup(v any) (string, bool) {
	n, ok := legacyInt(v)
	if !ok {
		return "", false
	}
	id, ok := m[n]
	return id, ok
}

func importLegacy(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := Now()
	exec := func(query string, args ...any) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}

	users := idMap{}
	rows, err := legacyRows(ctx, tx, "User", `SELECT * FROM "User"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		id, ok := users.assign(r["id"])
		if !ok {
			continue
		}
		role, _ := legacyStr(r["appRole"]).(string)
		switch role {
		case "ADMIN", "SECRETARY", "TREASURER", "READONLY":
		default:
			role = "READONLY"
		}
		created := legacyTimeOr(r["createdAt"], now)
		if err := exec(`INSERT INTO users (id, email, password, app_role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			id, legacyStr(r["email"]), legacyStr(r["password"]), role, created, legacyTimeOr(r["updatedAt"], created)); err != nil {
			return fmt.Errorf("users: %w", err)
		}
	}

	members := idMap{}
	rows, err = legacyRows(ctx, tx, "Member", `SELECT * FROM "Member"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		id, ok := members.assign(r["id"])
		if !ok {
			continue
		}
		number, _ := legacyInt(r["number"])
		if err := exec(`INSERT INTO members (id, number, first_name, last_name, birthday, entry_date, exit_date, comment, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, number, legacyStr(r["firstName"]), legacyStr(r["lastName"]),
			legacyTime(r["birthday"]), legacyTime(r["entryDate"]), legacyTime(r["exitDate"]), legacyStr(r["comment"]),
			now, now); err != nil {
			return fmt.Errorf("members: %w", err)
		}
		if err := exec(`INSERT INTO member_contacts (member_id, email, phone, phone_mobile, street, postal_code, city, state, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, legacyStr(r["email"]), legacyStr(r["phone"]), legacyStr(r["phoneMobile"]),
			legacyStr(r["street"]), legacyStr(r["postalCode"]), legacyStr(r["city"]), legacyStr(r["state"]), now); err != nil {
			return fmt.Errorf("member_contacts: %w", err)
		}
		bank := []any{legacyStr(r["accountHolder"]), legacyStr(r["iban"]), legacyStr(r["bic"]), legacyStr(r["bankName"]), legacyTime(r["sepaMandateDate"])}
		hasBank := false
		for _, v := range bank {
			if s, ok := v.(string); ok && s != "" {
				hasBank = true
			}
		}
		if hasBank {
			if err := exec(`INSERT INTO member_bank_details (member_id, account_holder, iban, bic, bank_name, sepa_mandate_date, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, append(append([]any{id}, bank...), now)...); err != nil {
				return fmt.Errorf("member_bank_details: %w", err)
			}
		}
	}

	// Named entities and their membership links. Prisma names the columns of
	// implicit m:n tables A and B in alphabetical model order, so members are
	// in A for _MemberRoles (Member < Role) but in B for _MemberGroups and
	// _MemberSections (Group/ClubSection < Member).
	named := []struct {
		legacy, table, junction, linkTable, linkCol, memberCol, entityCol string
	}{
		{"Role", "roles", "_MemberRoles", "member_roles", "role_id", "A", "B"},
		{"Group", "groups", "_MemberGroups", "member_groups", "group_id", "B", "A"},
		{"ClubSection", "sections", "_MemberSections", "member_sections", "section_id", "B", "A"},
	}
	entityIDs := map[string]idMap{"Member": members}
	for _, n := range named {
		ids := idMap{}
		entityIDs[n.legacy] = ids
		rows, err := legacyRows(ctx, tx, n.legacy, fmt.Sprintf(`SELECT * FROM "%s"`, n.legacy))
		if err != nil {
			return err
		}
		for _, r := range rows {
			id, ok := ids.assign(r["id"])
			if !ok {
				continue
			}
			if err := exec(fmt.Sprintf(`INSERT INTO %s (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`, n.table),
				id, legacyStr(r["name"]), now, now); err != nil {
				return fmt.Errorf("%s: %w", n.table, err)
			}
		}
		rows, err = legacyRows(ctx, tx, n.junction, fmt.Sprintf(`SELECT "A", "B" FROM "%s"`, n.junction))
		if err != nil {
			return err
		}
		for _, r := range rows {
			memberID, ok1 := members.lookup(r[n.memberCol])
			entityID, ok2 := ids.lookup(r[n.entityCol])
			if !ok1 || !ok2 {
				continue // dangling link
			}
			if err := exec(fmt.Sprintf(`INSERT OR IGNORE INTO %s (member_id, %s, updated_at) VALUES (?, ?, ?)`, n.linkTable, n.linkCol),
				memberID, entityID, now); err != nil {
				return fmt.Errorf("%s: %w", n.linkTable, err)
			}
		}
	}

	transactions := idMap{}
	rows, err = legacyRows(ctx, tx, "FinanceTransaction", `SELECT * FROM "FinanceTransaction"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		id, ok := transactions.assign(r["id"])
		if !ok {
			continue
		}
		amount, _ := legacyFloat(r["amount"])
		txType, _ := legacyStr(r["type"]).(string)
		if txType != "expense" {
			txType = "income"
		}
		created := legacyTimeOr(r["createdAt"], now)
		if err := exec(`INSERT INTO finance_transactions (id, date, description, amount_cents, type, category, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, legacyTimeOr(r["date"], created), legacyStr(r["description"]), ToCents(amount), txType,
			legacyStr(r["category"]), legacyStr(r["notes"]), created, legacyTimeOr(r["updatedAt"], created)); err != nil {
			return fmt.Errorf("finance_transactions: %w", err)
		}
	}

	fees := idMap{}
	rows, err = legacyRows(ctx, tx, "MemberFee", `SELECT * FROM "MemberFee"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		memberID, ok := members.lookup(r["memberId"])
		if !ok {
			continue
		}
		id, ok := fees.assign(r["id"])
		if !ok {
			continue
		}
		amount, _ := legacyFloat(r["amount"])
		year, _ := legacyInt(r["year"])
		created := legacyTimeOr(r["createdAt"], now)
		if err := exec(`INSERT INTO member_fees (id, member_id, amount_cents, due_date, paid_date, description, year, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, memberID, ToCents(amount), legacyTimeOr(r["dueDate"], created), legacyTime(r["paidDate"]),
			legacyStr(r["description"]), year, created, legacyTimeOr(r["updatedAt"], created)); err != nil {
			return fmt.Errorf("member_fees: %w", err)
		}
	}

	items := idMap{}
	rows, err = legacyRows(ctx, tx, "InventoryItem", `SELECT * FROM "InventoryItem"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		id, ok := items.assign(r["id"])
		if !ok {
			continue
		}
		quantity, _ := legacyInt(r["quantity"])
		var price any
		if p, ok := legacyFloat(r["purchasePrice"]); ok {
			price = ToCents(p)
		}
		created := legacyTimeOr(r["createdAt"], now)
		if err := exec(`INSERT INTO inventory_items (id, name, description, serial_number, category, quantity, location, purchase_date, purchase_price_cents, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, legacyStr(r["name"]), legacyStr(r["description"]), legacyStr(r["serialNumber"]), legacyStr(r["category"]),
			quantity, legacyStr(r["location"]), legacyTime(r["purchaseDate"]), price, created, legacyTimeOr(r["updatedAt"], created)); err != nil {
			return fmt.Errorf("inventory_items: %w", err)
		}
	}

	loans := idMap{}
	rows, err = legacyRows(ctx, tx, "InventoryLoan", `SELECT * FROM "InventoryLoan"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		itemID, ok1 := items.lookup(r["itemId"])
		memberID, ok2 := members.lookup(r["memberId"])
		if !ok1 || !ok2 {
			continue
		}
		id, ok := loans.assign(r["id"])
		if !ok {
			continue
		}
		notes, _ := legacyStr(r["notes"]).(string)
		created := legacyTimeOr(r["createdAt"], now)
		if err := exec(`INSERT INTO inventory_loans (id, item_id, member_id, loaned_at, due_date, returned_at, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, itemID, memberID, legacyTimeOr(r["loanedAt"], created), legacyTime(r["dueDate"]), legacyTime(r["returnedAt"]),
			notes, created, created); err != nil {
			return fmt.Errorf("inventory_loans: %w", err)
		}
	}

	rows, err = legacyRows(ctx, tx, "UserPreference", `SELECT * FROM "UserPreference"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		// The previous handlers stored every preference under user id 1, the
		// config-file admin.
		userID, ok := users.lookup(r["userId"])
		if n, _ := legacyInt(r["userId"]); n == 1 || !ok {
			userID = AdminUserID
		}
		if err := exec(`INSERT OR IGNORE INTO user_preferences (id, user_id, key, value, updated_at) VALUES (?, ?, ?, ?, ?)`,
			NewID(), userID, legacyStr(r["key"]), legacyStr(r["value"]), legacyTimeOr(r["updatedAt"], now)); err != nil {
			return fmt.Errorf("user_preferences: %w", err)
		}
	}

	entityIDs["FinanceTransaction"] = transactions
	entityIDs["MemberFee"] = fees
	entityIDs["InventoryItem"] = items
	entityIDs["InventoryLoan"] = loans
	rows, err = legacyRows(ctx, tx, "AuditLog", `SELECT * FROM "AuditLog"`)
	if err != nil {
		return err
	}
	for _, r := range rows {
		entity, _ := legacyStr(r["entity"]).(string)
		entityID := ""
		if m, ok := entityIDs[entity]; ok {
			entityID, _ = m.lookup(r["entityId"])
		}
		if entityID == "" {
			n, _ := legacyInt(r["entityId"])
			entityID = "legacy:" + strconv.FormatInt(n, 10)
		}
		action, _ := legacyStr(r["action"]).(string)
		switch action {
		case "CREATE", "UPDATE", "DELETE":
		default:
			continue
		}
		if err := exec(`INSERT INTO audit_log (id, action, entity, entity_id, user_id, data, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			NewID(), action, entity, entityID, legacyStr(r["userId"]), legacyStr(r["data"]), legacyTimeOr(r["createdAt"], now)); err != nil {
			return fmt.Errorf("audit_log: %w", err)
		}
	}

	for _, t := range legacyTables {
		if err := exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, t)); err != nil {
			return fmt.Errorf("drop legacy table %s: %w", t, err)
		}
	}
	return tx.Commit()
}
