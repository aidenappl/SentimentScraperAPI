package structs

type Company struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Ticker     string  `json:"ticker"`
	Website    *string `json:"website"`
	Logo       *string `json:"logo"`
	InsertedAt string  `json:"inserted_at"`
	CIK        *string `json:"cik"`
}
