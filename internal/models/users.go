package models

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"zedproxy/internal/database"
)

// ─── Helpers ─────────────────────────────────────────

func HashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func GeneratePublicID(prefix string) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return prefix + string(b)
}

func GenerateToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ─── User ─────────────────────────────────────────────

type User struct {
	ID                  int64
	PublicID            string
	Email               sql.NullString
	Phone               sql.NullString
	PasswordHash        string
	Status              string
	Role                string
	TelegramID          sql.NullString
	TelegramUsername    sql.NullString
	TelegramConnectedAt sql.NullTime
	EmailVerifiedAt     sql.NullTime
	PhoneVerifiedAt     sql.NullTime
	LastLoginAt         sql.NullTime
	LastLoginIPHash     sql.NullString
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           sql.NullTime
	// Joined
	Profile             *UserProfile
	WalletBalance       int64
	UnreadNotifications int
	OpenTickets         int
}

func (u *User) DisplayEmail() string {
	if u.Email.Valid {
		return u.Email.String
	}
	return ""
}

func (u *User) DisplayPhone() string {
	if u.Phone.Valid {
		return u.Phone.String
	}
	return ""
}

func (u *User) DisplayName() string {
	if u.Profile != nil {
		if u.Profile.DisplayName != "" {
			return u.Profile.DisplayName
		}
		if u.Profile.FirstName != "" {
			name := u.Profile.FirstName
			if u.Profile.LastName != "" {
				name += " " + u.Profile.LastName
			}
			return name
		}
	}
	if u.Email.Valid {
		parts := strings.Split(u.Email.String, "@")
		return parts[0]
	}
	if u.Phone.Valid {
		p := u.Phone.String
		if len(p) > 4 {
			return p[:4] + "****"
		}
		return p
	}
	return "کاربر"
}

func (u *User) TelegramConnected() bool {
	return u.TelegramID.Valid && u.TelegramID.String != ""
}

func (u *User) IsBlocked() bool {
	return u.Status == "blocked"
}

var userCols = `id, public_id, COALESCE(email,''), COALESCE(phone,''), password_hash, status, role,
	COALESCE(telegram_id,''), COALESCE(telegram_username,''),
	telegram_connected_at, email_verified_at, phone_verified_at, last_login_at,
	COALESCE(last_login_ip_hash,''), created_at, updated_at, deleted_at`

func scanUser(row interface{ Scan(...interface{}) error }) (*User, error) {
	u := &User{}
	var email, phone, tgID, tgUser, lastIPHash string
	var tgAt, emailVerAt, phoneVerAt, lastLoginAt, deletedAt sql.NullTime
	var updatedAt, createdAt time.Time
	err := row.Scan(
		&u.ID, &u.PublicID, &email, &phone, &u.PasswordHash, &u.Status, &u.Role,
		&tgID, &tgUser, &tgAt, &emailVerAt, &phoneVerAt, &lastLoginAt,
		&lastIPHash, &createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}
	if email != "" {
		u.Email = sql.NullString{String: email, Valid: true}
	}
	if phone != "" {
		u.Phone = sql.NullString{String: phone, Valid: true}
	}
	if tgID != "" {
		u.TelegramID = sql.NullString{String: tgID, Valid: true}
	}
	if tgUser != "" {
		u.TelegramUsername = sql.NullString{String: tgUser, Valid: true}
	}
	if lastIPHash != "" {
		u.LastLoginIPHash = sql.NullString{String: lastIPHash, Valid: true}
	}
	u.TelegramConnectedAt = tgAt
	u.EmailVerifiedAt = emailVerAt
	u.PhoneVerifiedAt = phoneVerAt
	u.LastLoginAt = lastLoginAt
	u.DeletedAt = deletedAt
	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt
	return u, nil
}

func CreateUser(email, phone, passwordHash string) (*User, error) {
	publicID := GeneratePublicID("usr_")
	now := time.Now()
	var emailVal, phoneVal interface{}
	if email != "" {
		emailVal = email
	}
	if phone != "" {
		phoneVal = phone
	}
	res, err := database.DB.Exec(
		`INSERT INTO users (public_id, email, phone, password_hash, status, role, created_at, updated_at)
		 VALUES (?,?,?,?,'active','user',?,?)`,
		publicID, emailVal, phoneVal, passwordHash, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	database.DB.Exec(
		`INSERT INTO user_profiles (user_id, timezone, created_at, updated_at) VALUES (?,?,?,?)`,
		id, "Asia/Tehran", now, now,
	)
	return GetUserByID(id)
}

func GetUserByID(id int64) (*User, error) {
	row := database.DB.QueryRow(
		`SELECT `+userCols+` FROM users WHERE id = ? AND deleted_at IS NULL`, id,
	)
	u, err := scanUser(row)
	if err != nil {
		return nil, err
	}
	u.Profile, _ = GetUserProfile(id)
	return u, nil
}

func GetUserByPublicID(publicID string) (*User, error) {
	row := database.DB.QueryRow(
		`SELECT `+userCols+` FROM users WHERE public_id = ? AND deleted_at IS NULL`, publicID,
	)
	u, err := scanUser(row)
	if err != nil {
		return nil, err
	}
	u.Profile, _ = GetUserProfile(u.ID)
	return u, nil
}

func GetUserByEmail(email string) (*User, error) {
	row := database.DB.QueryRow(
		`SELECT `+userCols+` FROM users WHERE email = ? AND deleted_at IS NULL`, strings.ToLower(email),
	)
	return scanUser(row)
}

func GetUserByPhone(phone string) (*User, error) {
	row := database.DB.QueryRow(
		`SELECT `+userCols+` FROM users WHERE phone = ? AND deleted_at IS NULL`, phone,
	)
	return scanUser(row)
}

func GetUserByEmailOrPhone(identifier string) (*User, error) {
	row := database.DB.QueryRow(
		`SELECT `+userCols+` FROM users WHERE (email = ? OR phone = ?) AND deleted_at IS NULL`,
		strings.ToLower(identifier), identifier,
	)
	return scanUser(row)
}

func GetUserByTelegramID(telegramID string) (*User, error) {
	row := database.DB.QueryRow(
		`SELECT `+userCols+` FROM users WHERE telegram_id = ? AND deleted_at IS NULL`, telegramID,
	)
	return scanUser(row)
}

func EmailExists(email string) bool {
	var id int64
	database.DB.QueryRow(`SELECT id FROM users WHERE email = ? AND deleted_at IS NULL`, strings.ToLower(email)).Scan(&id)
	return id > 0
}

func PhoneExists(phone string) bool {
	var id int64
	database.DB.QueryRow(`SELECT id FROM users WHERE phone = ? AND deleted_at IS NULL`, phone).Scan(&id)
	return id > 0
}

func UpdateUserLastLogin(userID int64, ipHash string) {
	database.DB.Exec(
		`UPDATE users SET last_login_at = ?, last_login_ip_hash = ?, updated_at = ? WHERE id = ?`,
		time.Now(), ipHash, time.Now(), userID,
	)
}

func UpdateUserStatus(userID int64, status string) error {
	_, err := database.DB.Exec(
		`UPDATE users SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now(), userID,
	)
	return err
}

func UpdateUserPassword(userID int64, passwordHash string) error {
	_, err := database.DB.Exec(
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, passwordHash, time.Now(), userID,
	)
	return err
}

func ConnectUserTelegram(userID int64, telegramID, telegramUsername string) error {
	_, err := database.DB.Exec(
		`UPDATE users SET telegram_id = ?, telegram_username = ?, telegram_connected_at = ?, updated_at = ? WHERE id = ?`,
		telegramID, telegramUsername, time.Now(), time.Now(), userID,
	)
	return err
}

func DisconnectUserTelegram(userID int64) error {
	_, err := database.DB.Exec(
		`UPDATE users SET telegram_id = NULL, telegram_username = NULL, telegram_connected_at = NULL, updated_at = ? WHERE id = ?`,
		time.Now(), userID,
	)
	return err
}

type UserListFilter struct {
	Search    string
	Status    string
	HasTG     string // "yes" | "no" | ""
	Page      int
	PageSize  int
}

type UserListResult struct {
	Users   []*User
	Total   int
	Page    int
	Pages   int
}

func ListUsers(f UserListFilter) (*UserListResult, error) {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}

	where := "WHERE u.deleted_at IS NULL"
	args := []interface{}{}

	if f.Search != "" {
		where += " AND (u.email LIKE ? OR u.phone LIKE ? OR u.telegram_username LIKE ? OR p.display_name LIKE ?)"
		s := "%" + f.Search + "%"
		args = append(args, s, s, s, s)
	}
	if f.Status != "" {
		where += " AND u.status = ?"
		args = append(args, f.Status)
	}
	switch f.HasTG {
	case "yes":
		where += " AND u.telegram_id IS NOT NULL AND u.telegram_id != ''"
	case "no":
		where += " AND (u.telegram_id IS NULL OR u.telegram_id = '')"
	}

	countQ := `SELECT COUNT(*) FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id ` + where
	var total int
	if err := database.DB.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (f.Page - 1) * f.PageSize
	q := `SELECT ` + userCols + ` FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id ` +
		where + ` ORDER BY u.created_at DESC LIMIT ? OFFSET ?`
	args = append(args, f.PageSize, offset)

	rows, err := database.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			continue
		}
		u.Profile, _ = GetUserProfile(u.ID)
		users = append(users, u)
	}

	pages := (total + f.PageSize - 1) / f.PageSize
	return &UserListResult{Users: users, Total: total, Page: f.Page, Pages: pages}, nil
}

// ─── UserProfile ─────────────────────────────────────

type UserProfile struct {
	ID            int64
	UserID        int64
	FirstName     string
	LastName      string
	DisplayName   string
	Timezone      string
	Country       string
	PrimaryDevice string
	UsageType     string
	AvatarPath    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func GetUserProfile(userID int64) (*UserProfile, error) {
	p := &UserProfile{}
	err := database.DB.QueryRow(
		`SELECT id, user_id, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(display_name,''),
		 COALESCE(timezone,'Asia/Tehran'), COALESCE(country,''), COALESCE(primary_device,''),
		 COALESCE(usage_type,''), COALESCE(avatar_path,''), created_at, updated_at
		 FROM user_profiles WHERE user_id = ?`, userID,
	).Scan(&p.ID, &p.UserID, &p.FirstName, &p.LastName, &p.DisplayName,
		&p.Timezone, &p.Country, &p.PrimaryDevice, &p.UsageType, &p.AvatarPath,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func UpsertUserProfile(userID int64, first, last, display, timezone, country, device, usage string) error {
	now := time.Now()
	_, err := database.DB.Exec(
		`INSERT INTO user_profiles (user_id, first_name, last_name, display_name, timezone, country, primary_device, usage_type, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(user_id) DO UPDATE SET first_name=excluded.first_name, last_name=excluded.last_name,
		 display_name=excluded.display_name, timezone=excluded.timezone, country=excluded.country,
		 primary_device=excluded.primary_device, usage_type=excluded.usage_type, updated_at=excluded.updated_at`,
		userID, nvl(first), nvl(last), nvl(display), nvl(timezone), nvl(country), nvl(device), nvl(usage), now, now,
	)
	database.DB.Exec(`UPDATE users SET updated_at=? WHERE id=?`, now, userID)
	return err
}

func UpdateUserContact(userID int64, email, phone string) error {
	now := time.Now()
	var err error
	if email != "" {
		_, err = database.DB.Exec(`UPDATE users SET email=?, updated_at=? WHERE id=?`, strings.ToLower(email), now, userID)
	}
	if phone != "" && err == nil {
		_, err = database.DB.Exec(`UPDATE users SET phone=?, updated_at=? WHERE id=?`, phone, now, userID)
	}
	return err
}

func nvl(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ─── Password Reset ───────────────────────────────────

type PasswordResetToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    sql.NullTime
}

func CreatePasswordResetToken(userID int64) (string, error) {
	raw := GenerateToken(24)
	hash := HashString(raw)
	exp := time.Now().Add(1 * time.Hour)
	database.DB.Exec(`UPDATE password_reset_tokens SET used_at=? WHERE user_id=? AND used_at IS NULL`, time.Now(), userID)
	_, err := database.DB.Exec(
		`INSERT INTO password_reset_tokens (user_id, token_hash, created_at, expires_at) VALUES (?,?,?,?)`,
		userID, hash, time.Now(), exp,
	)
	if err != nil {
		return "", err
	}
	return raw, nil
}

func ValidatePasswordResetToken(rawToken string) (int64, error) {
	hash := HashString(rawToken)
	var t PasswordResetToken
	var usedAt sql.NullTime
	err := database.DB.QueryRow(
		`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = ?`, hash,
	).Scan(&t.ID, &t.UserID, &t.ExpiresAt, &usedAt)
	if err != nil {
		return 0, fmt.Errorf("invalid token")
	}
	if usedAt.Valid {
		return 0, fmt.Errorf("token already used")
	}
	if time.Now().After(t.ExpiresAt) {
		return 0, fmt.Errorf("token expired")
	}
	database.DB.Exec(`UPDATE password_reset_tokens SET used_at=? WHERE id=?`, time.Now(), t.ID)
	return t.UserID, nil
}

// ─── Telegram Connect Token ───────────────────────────

type TelegramConnectToken struct {
	ID          int64
	UserID      int64
	TokenPublic string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

func CreateTelegramConnectToken(userID int64) (string, error) {
	raw := GenerateToken(16)
	hash := HashString(raw)
	exp := time.Now().Add(30 * time.Minute)
	database.DB.Exec(`UPDATE telegram_connect_tokens SET used_at=? WHERE user_id=? AND used_at IS NULL`, time.Now(), userID)
	_, err := database.DB.Exec(
		`INSERT INTO telegram_connect_tokens (user_id, token_hash, token_public, created_at, expires_at) VALUES (?,?,?,?,?)`,
		userID, hash, raw, time.Now(), exp,
	)
	if err != nil {
		return "", err
	}
	return raw, nil
}

func ValidateTelegramConnectToken(rawToken string) (int64, error) {
	hash := HashString(rawToken)
	var userID int64
	var exp time.Time
	var usedAt sql.NullTime
	err := database.DB.QueryRow(
		`SELECT user_id, expires_at, used_at FROM telegram_connect_tokens WHERE token_hash = ?`, hash,
	).Scan(&userID, &exp, &usedAt)
	if err != nil {
		return 0, fmt.Errorf("invalid token")
	}
	if usedAt.Valid {
		return 0, fmt.Errorf("token already used")
	}
	if time.Now().After(exp) {
		return 0, fmt.Errorf("token expired")
	}
	database.DB.Exec(`UPDATE telegram_connect_tokens SET used_at=? WHERE token_hash=?`, time.Now(), hash)
	return userID, nil
}

// ─── UserService ─────────────────────────────────────

type UserService struct {
	ID                    int64
	UserID                int64
	Title                 string
	PlanName              string
	Status                string
	TrafficTotalBytes     int64
	TrafficUsedBytes      int64
	TrafficRemainingBytes int64
	StartedAt             sql.NullTime
	ExpiresAt             sql.NullTime
	Location              string
	SubscriptionURL       string
	QRCodePath            string
	Source                string
	ExternalServiceID     string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (s *UserService) TrafficTotalGB() float64 { return float64(s.TrafficTotalBytes) / 1073741824 }
func (s *UserService) TrafficUsedGB() float64  { return float64(s.TrafficUsedBytes) / 1073741824 }
func (s *UserService) TrafficPercent() int {
	if s.TrafficTotalBytes == 0 {
		return 0
	}
	p := int(s.TrafficUsedBytes * 100 / s.TrafficTotalBytes)
	if p > 100 {
		return 100
	}
	return p
}
func (s *UserService) DaysRemaining() int {
	if !s.ExpiresAt.Valid {
		return 0
	}
	d := time.Until(s.ExpiresAt.Time)
	if d < 0 {
		return 0
	}
	return int(d.Hours() / 24)
}

func scanUserService(row interface{ Scan(...interface{}) error }) (*UserService, error) {
	s := &UserService{}
	err := row.Scan(
		&s.ID, &s.UserID, &s.Title, &s.PlanName, &s.Status,
		&s.TrafficTotalBytes, &s.TrafficUsedBytes, &s.TrafficRemainingBytes,
		&s.StartedAt, &s.ExpiresAt, &s.Location, &s.SubscriptionURL,
		&s.QRCodePath, &s.Source, &s.ExternalServiceID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}

var svcCols = `id, user_id, title, COALESCE(plan_name,''), status,
	traffic_total_bytes, traffic_used_bytes, traffic_remaining_bytes,
	started_at, expires_at, COALESCE(location,''), COALESCE(subscription_url,''),
	COALESCE(qr_code_path,''), COALESCE(source,'manual'), COALESCE(external_service_id,''),
	created_at, updated_at`

func GetUserServices(userID int64) ([]*UserService, error) {
	rows, err := database.DB.Query(
		`SELECT `+svcCols+` FROM user_services WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*UserService
	for rows.Next() {
		s, err := scanUserService(rows)
		if err == nil {
			list = append(list, s)
		}
	}
	return list, nil
}

func GetUserServiceByID(id, userID int64) (*UserService, error) {
	row := database.DB.QueryRow(
		`SELECT `+svcCols+` FROM user_services WHERE id = ? AND user_id = ?`, id, userID,
	)
	return scanUserService(row)
}

func GetActiveUserService(userID int64) (*UserService, error) {
	row := database.DB.QueryRow(
		`SELECT `+svcCols+` FROM user_services WHERE user_id = ? AND status = 'active' ORDER BY expires_at DESC LIMIT 1`, userID,
	)
	return scanUserService(row)
}

func CreateUserService(userID int64, title, plan, status, location, subURL string, totalBytes int64, startAt, expiresAt *time.Time) error {
	now := time.Now()
	_, err := database.DB.Exec(
		`INSERT INTO user_services (user_id, title, plan_name, status, traffic_total_bytes, traffic_remaining_bytes,
		 location, subscription_url, source, started_at, expires_at, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,'manual',?,?,?,?)`,
		userID, title, nvl(plan), status, totalBytes, totalBytes, nvl(location), nvl(subURL),
		timeOrNull(startAt), timeOrNull(expiresAt), now, now,
	)
	return err
}

func timeOrNull(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

// ─── UserOrder ────────────────────────────────────────

type UserOrder struct {
	ID                 int64
	UserID             int64
	OrderNumber        string
	PlanName           string
	Amount             int64
	Currency           string
	PaymentMethod      string
	PaymentStatus      string
	OrderStatus        string
	DiscountCode       string
	Source             string
	TelegramStartParam string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

var orderCols = `id, user_id, order_number, COALESCE(plan_name,''), amount, currency,
	COALESCE(payment_method,''), payment_status, order_status,
	COALESCE(discount_code,''), COALESCE(source,''), COALESCE(telegram_start_param,''),
	created_at, updated_at`

func scanUserOrder(row interface{ Scan(...interface{}) error }) (*UserOrder, error) {
	o := &UserOrder{}
	err := row.Scan(
		&o.ID, &o.UserID, &o.OrderNumber, &o.PlanName, &o.Amount, &o.Currency,
		&o.PaymentMethod, &o.PaymentStatus, &o.OrderStatus,
		&o.DiscountCode, &o.Source, &o.TelegramStartParam,
		&o.CreatedAt, &o.UpdatedAt,
	)
	return o, err
}

func GetUserOrders(userID int64) ([]*UserOrder, error) {
	rows, err := database.DB.Query(
		`SELECT `+orderCols+` FROM user_orders WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*UserOrder
	for rows.Next() {
		o, err := scanUserOrder(rows)
		if err == nil {
			list = append(list, o)
		}
	}
	return list, nil
}

func GetUserOrderByNumber(orderNumber string, userID int64) (*UserOrder, error) {
	row := database.DB.QueryRow(
		`SELECT `+orderCols+` FROM user_orders WHERE order_number = ? AND user_id = ?`, orderNumber, userID,
	)
	return scanUserOrder(row)
}

// ─── Wallet ───────────────────────────────────────────

type WalletTransaction struct {
	ID               int64
	UserID           int64
	Type             string
	Amount           int64
	Currency         string
	BalanceAfter     int64
	Description      string
	ReferenceType    string
	ReferenceID      string
	CreatedByAdminID sql.NullInt64
	CreatedAt        time.Time
	// Joined
	AdminUsername string
}

func GetWalletBalance(userID int64) int64 {
	var bal int64
	database.DB.QueryRow(
		`SELECT COALESCE(balance_after,0) FROM wallet_transactions WHERE user_id = ? ORDER BY id DESC LIMIT 1`, userID,
	).Scan(&bal)
	return bal
}

func GetWalletTransactions(userID int64) ([]*WalletTransaction, error) {
	rows, err := database.DB.Query(
		`SELECT t.id, t.user_id, t.type, t.amount, t.currency, t.balance_after,
		 COALESCE(t.description,''), COALESCE(t.reference_type,''), COALESCE(t.reference_id,''),
		 t.created_by_admin_id, t.created_at, COALESCE(a.username,'')
		 FROM wallet_transactions t
		 LEFT JOIN admins a ON a.id = t.created_by_admin_id
		 WHERE t.user_id = ? ORDER BY t.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*WalletTransaction
	for rows.Next() {
		tx := &WalletTransaction{}
		rows.Scan(
			&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.BalanceAfter,
			&tx.Description, &tx.ReferenceType, &tx.ReferenceID,
			&tx.CreatedByAdminID, &tx.CreatedAt, &tx.AdminUsername,
		)
		list = append(list, tx)
	}
	return list, nil
}

func AdjustWallet(userID, adminID int64, txType string, amount int64, description string) error {
	bal := GetWalletBalance(userID)
	switch txType {
	case "credit", "gift", "refund", "adjustment":
		bal += amount
	case "debit":
		bal -= amount
	}
	_, err := database.DB.Exec(
		`INSERT INTO wallet_transactions (user_id, type, amount, currency, balance_after, description, created_by_admin_id, created_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		userID, txType, amount, "IRT", bal, nvl(description), func() interface{} {
			if adminID > 0 {
				return adminID
			}
			return nil
		}(), time.Now(),
	)
	return err
}

// ─── Support Tickets ─────────────────────────────────

type SupportTicket struct {
	ID              int64
	UserID          int64
	TicketNumber    string
	Subject         string
	Category        string
	Priority        string
	Status          string
	LastMessageAt   sql.NullTime
	AssignedAdminID sql.NullInt64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ClosedAt        sql.NullTime
	// Joined
	User          *User
	Messages      []*SupportTicketMessage
	MessageCount  int
}

type SupportTicketMessage struct {
	ID             int64
	TicketID       int64
	SenderType     string
	SenderUserID   sql.NullInt64
	SenderAdminID  sql.NullInt64
	Message        string
	AttachmentPath string
	CreatedAt      time.Time
	// Joined
	SenderName string
}

var ticketCols = `id, user_id, ticket_number, subject, category, priority, status,
	last_message_at, assigned_admin_id, created_at, updated_at, closed_at`

func scanTicket(row interface{ Scan(...interface{}) error }) (*SupportTicket, error) {
	t := &SupportTicket{}
	err := row.Scan(
		&t.ID, &t.UserID, &t.TicketNumber, &t.Subject, &t.Category, &t.Priority, &t.Status,
		&t.LastMessageAt, &t.AssignedAdminID, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt,
	)
	return t, err
}

func GenerateTicketNumber() string {
	raw := GenerateToken(4)
	return fmt.Sprintf("TKT-%s", strings.ToUpper(hex.EncodeToString([]byte(raw))[:8]))
}

func CreateSupportTicket(userID int64, subject, category, message string) (*SupportTicket, error) {
	number := GenerateTicketNumber()
	now := time.Now()
	res, err := database.DB.Exec(
		`INSERT INTO support_tickets (user_id, ticket_number, subject, category, priority, status, last_message_at, created_at, updated_at)
		 VALUES (?,?,?,?,'normal','open',?,?,?)`,
		userID, number, subject, category, now, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	database.DB.Exec(
		`INSERT INTO support_ticket_messages (ticket_id, sender_type, sender_user_id, message, created_at)
		 VALUES (?,'user',?,?,?)`, id, userID, message, now,
	)
	return GetTicketByNumber(number, userID)
}

func GetUserTickets(userID int64) ([]*SupportTicket, error) {
	rows, err := database.DB.Query(
		`SELECT `+ticketCols+` FROM support_tickets WHERE user_id = ? ORDER BY updated_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*SupportTicket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err == nil {
			list = append(list, t)
		}
	}
	return list, nil
}

func GetTicketByNumber(number string, userID int64) (*SupportTicket, error) {
	q := `SELECT ` + ticketCols + ` FROM support_tickets WHERE ticket_number = ?`
	args := []interface{}{number}
	if userID > 0 {
		q += " AND user_id = ?"
		args = append(args, userID)
	}
	row := database.DB.QueryRow(q, args...)
	t, err := scanTicket(row)
	if err != nil {
		return nil, err
	}
	t.Messages, _ = GetTicketMessages(t.ID)
	t.User, _ = GetUserByID(t.UserID)
	return t, nil
}

func GetTicketMessages(ticketID int64) ([]*SupportTicketMessage, error) {
	rows, err := database.DB.Query(
		`SELECT m.id, m.ticket_id, m.sender_type, m.sender_user_id, m.sender_admin_id,
		 m.message, COALESCE(m.attachment_path,''), m.created_at,
		 COALESCE(u.email, COALESCE(p.display_name, '')), COALESCE(a.username,'')
		 FROM support_ticket_messages m
		 LEFT JOIN users u ON u.id = m.sender_user_id
		 LEFT JOIN user_profiles p ON p.user_id = m.sender_user_id
		 LEFT JOIN admins a ON a.id = m.sender_admin_id
		 WHERE m.ticket_id = ? ORDER BY m.created_at ASC`, ticketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*SupportTicketMessage
	for rows.Next() {
		m := &SupportTicketMessage{}
		var userContact, adminName string
		rows.Scan(
			&m.ID, &m.TicketID, &m.SenderType, &m.SenderUserID, &m.SenderAdminID,
			&m.Message, &m.AttachmentPath, &m.CreatedAt,
			&userContact, &adminName,
		)
		if m.SenderType == "admin" {
			m.SenderName = adminName
			if m.SenderName == "" {
				m.SenderName = "پشتیبانی"
			}
		} else {
			m.SenderName = userContact
			if m.SenderName == "" {
				m.SenderName = "کاربر"
			}
		}
		list = append(list, m)
	}
	return list, nil
}

func AddTicketMessage(ticketID, senderUserID, senderAdminID int64, senderType, message string) error {
	now := time.Now()
	var userArg, adminArg interface{}
	if senderUserID > 0 {
		userArg = senderUserID
	}
	if senderAdminID > 0 {
		adminArg = senderAdminID
	}
	_, err := database.DB.Exec(
		`INSERT INTO support_ticket_messages (ticket_id, sender_type, sender_user_id, sender_admin_id, message, created_at)
		 VALUES (?,?,?,?,?,?)`, ticketID, senderType, userArg, adminArg, message, now,
	)
	if err != nil {
		return err
	}
	database.DB.Exec(
		`UPDATE support_tickets SET last_message_at=?, updated_at=? WHERE id=?`, now, now, ticketID,
	)
	return nil
}

func UpdateTicketStatus(ticketID int64, status string) error {
	now := time.Now()
	var closedAt interface{}
	if status == "closed" {
		closedAt = now
	}
	_, err := database.DB.Exec(
		`UPDATE support_tickets SET status=?, closed_at=?, updated_at=? WHERE id=?`,
		status, closedAt, now, ticketID,
	)
	return err
}

func OpenTicketCount(userID int64) int {
	var n int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM support_tickets WHERE user_id=? AND status != 'closed'`, userID,
	).Scan(&n)
	return n
}

type TicketListFilter struct {
	// Status values: "open", "closed", "waiting_admin", "waiting_user",
	// "not_closed" (any non-closed), "pending_admin" (alias waiting_admin),
	// "answered" (alias waiting_user)
	Status   string
	Category string
	Search   string // search by ticket number, user id, subject, email
	Page     int
	PageSize int
}

func ListAllTickets(f TicketListFilter) ([]*SupportTicket, int, error) {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	where := "WHERE 1=1"
	args := []interface{}{}

	switch f.Status {
	case "not_closed":
		where += " AND status != 'closed'"
	case "pending_admin":
		where += " AND status = 'waiting_admin'"
	case "answered":
		where += " AND status = 'waiting_user'"
	case "":
		// no filter
	default:
		where += " AND status=?"
		args = append(args, f.Status)
	}

	if f.Category != "" {
		where += " AND category=?"
		args = append(args, f.Category)
	}
	if f.Search != "" {
		s := "%" + f.Search + "%"
		where += ` AND (ticket_number LIKE ? OR subject LIKE ? OR CAST(user_id AS TEXT) LIKE ?
			OR EXISTS (SELECT 1 FROM users u WHERE u.id = support_tickets.user_id AND (u.email LIKE ? OR u.phone LIKE ?)))`
		args = append(args, s, s, s, s, s)
	}

	var total int
	database.DB.QueryRow("SELECT COUNT(*) FROM support_tickets "+where, args...).Scan(&total)
	offset := (f.Page - 1) * f.PageSize
	rows, err := database.DB.Query(
		`SELECT `+ticketCols+` FROM support_tickets `+where+
			` ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		append(args, f.PageSize, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []*SupportTicket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err == nil {
			t.User, _ = GetUserByID(t.UserID)
			list = append(list, t)
		}
	}
	return list, total, nil
}

// ─── Notifications ────────────────────────────────────

type UserNotification struct {
	ID        int64
	UserID    int64
	Title     string
	Message   string
	Type      string
	LinkURL   string
	ReadAt    sql.NullTime
	CreatedAt time.Time
}

func (n *UserNotification) IsRead() bool { return n.ReadAt.Valid }

func CreateNotification(userID int64, title, message, nType, linkURL string) error {
	_, err := database.DB.Exec(
		`INSERT INTO user_notifications (user_id, title, message, type, link_url, created_at)
		 VALUES (?,?,?,?,?,?)`,
		userID, title, message, nType, nvl(linkURL), time.Now(),
	)
	return err
}

func GetUserNotifications(userID int64) ([]*UserNotification, error) {
	rows, err := database.DB.Query(
		`SELECT id, user_id, title, message, type, COALESCE(link_url,''), read_at, created_at
		 FROM user_notifications WHERE user_id=? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*UserNotification
	for rows.Next() {
		n := &UserNotification{}
		rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.LinkURL, &n.ReadAt, &n.CreatedAt)
		list = append(list, n)
	}
	return list, nil
}

func GetNotificationByID(id, userID int64) (*UserNotification, error) {
	n := &UserNotification{}
	err := database.DB.QueryRow(
		`SELECT id, user_id, title, message, type, COALESCE(link_url,''), read_at, created_at
		 FROM user_notifications WHERE id=? AND user_id=?`, id, userID,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.LinkURL, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func MarkNotificationRead(id, userID int64) error {
	_, err := database.DB.Exec(
		`UPDATE user_notifications SET read_at=? WHERE id=? AND user_id=? AND read_at IS NULL`,
		time.Now(), id, userID,
	)
	return err
}

func UnreadNotificationCount(userID int64) int {
	var n int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM user_notifications WHERE user_id=? AND read_at IS NULL`, userID,
	).Scan(&n)
	return n
}

// ─── Activity Log ─────────────────────────────────────

type UserActivityLog struct {
	ID          int64
	UserID      int64
	Action      string
	Description string
	IPHash      string
	UserAgent   string
	CreatedAt   time.Time
}

func LogUserActivity(userID int64, action, description, ipHash, userAgent string) {
	database.DB.Exec(
		`INSERT INTO user_activity_logs (user_id, action, description, ip_hash, user_agent, created_at)
		 VALUES (?,?,?,?,?,?)`,
		userID, action, nvl(description), nvl(ipHash), nvl(userAgent), time.Now(),
	)
}

func GetUserActivityLogs(userID int64, limit int) ([]*UserActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := database.DB.Query(
		`SELECT id, user_id, action, COALESCE(description,''), COALESCE(ip_hash,''), COALESCE(user_agent,''), created_at
		 FROM user_activity_logs WHERE user_id=? ORDER BY created_at DESC LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*UserActivityLog
	for rows.Next() {
		l := &UserActivityLog{}
		rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Description, &l.IPHash, &l.UserAgent, &l.CreatedAt)
		list = append(list, l)
	}
	return list, nil
}

// RevokeAllUserSessions marks all active sessions revoked in DB (best-effort).
func RevokeAllUserSessions(userID int64) {
	database.DB.Exec(`UPDATE user_sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, time.Now(), userID)
}

// ─── Admin User Notes ─────────────────────────────────

type AdminUserNote struct {
	ID            int64
	UserID        int64
	AdminID       int64
	Note          string
	CreatedAt     time.Time
	AdminUsername string
}

func AddUserNote(userID, adminID int64, note string) error {
	_, err := database.DB.Exec(
		`INSERT INTO admin_user_notes (user_id, admin_id, note, created_at) VALUES (?,?,?,?)`,
		userID, adminID, note, time.Now(),
	)
	return err
}

func GetUserNotes(userID int64) ([]*AdminUserNote, error) {
	rows, err := database.DB.Query(
		`SELECT n.id, n.user_id, n.admin_id, n.note, n.created_at, COALESCE(a.username,'')
		 FROM admin_user_notes n LEFT JOIN admins a ON a.id=n.admin_id
		 WHERE n.user_id=? ORDER BY n.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*AdminUserNote
	for rows.Next() {
		n := &AdminUserNote{}
		rows.Scan(&n.ID, &n.UserID, &n.AdminID, &n.Note, &n.CreatedAt, &n.AdminUsername)
		list = append(list, n)
	}
	return list, nil
}
