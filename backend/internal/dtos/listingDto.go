package dtos

// --- Request DTOs ---

type CreateListingInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Quantity    int32   `json:"quantity"`
	Unit        string  `json:"unit"`
}

type UpdateListingInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Quantity    int32   `json:"quantity"`
	Unit        string  `json:"unit"`
}