package structs

import "time"

type Outlet struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Website    *string   `json:"website"`
	Logo       *string   `json:"logo"`
	InsertedAt time.Time `json:"inserted_at"`
}
