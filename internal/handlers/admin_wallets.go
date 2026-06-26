package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"zedproxy/internal/database"
)

type WalletUser struct {
	PublicID        string
	Email           string
	Phone           string
	WalletBalance   int64
	LastTxAmount    int64
	LastTxType      string
	LastTxCreatedAt time.Time
	HasLastTx       bool
}

func AdminWalletsPage(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT u.public_id, COALESCE(u.email,''), COALESCE(u.phone,''),
		COALESCE((SELECT SUM(CASE WHEN type='credit' THEN amount ELSE -amount END) FROM wallet_transactions WHERE user_id=u.id),0),
		COALESCE(t.amount,0), COALESCE(t.type,''), COALESCE(t.created_at, u.created_at)
		FROM users u
		LEFT JOIN wallet_transactions t ON t.id = (
			SELECT id FROM wallet_transactions WHERE user_id=u.id ORDER BY created_at DESC LIMIT 1
		)
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
		LIMIT 200`)

	var users []WalletUser
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var w WalletUser
			rows.Scan(&w.PublicID, &w.Email, &w.Phone, &w.WalletBalance,
				&w.LastTxAmount, &w.LastTxType, &w.LastTxCreatedAt)
			if w.LastTxType != "" {
				w.HasLastTx = true
			}
			users = append(users, w)
		}
	}

	data := adminData(c, "wallets")
	data["Title"] = "کیف پول کاربران"
	data["Users"] = users
	renderAdmin(c, "wallets", data)
}
