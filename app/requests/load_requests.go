package requests

type LoadStoreRequest struct {
	Title                string  `form:"title"                  json:"title"`
	Description          string  `form:"description"            json:"description"`
	SpecialInstructions  string  `form:"special_instructions"   json:"special_instructions"`

	PickupDestinationID  uint    `form:"pickup_destination_id"  json:"pickup_destination_id"`
	DropoffDestinationID uint    `form:"dropoff_destination_id" json:"dropoff_destination_id"`

	PickupDate  string `form:"pickup_date"  json:"pickup_date"`
	DropoffDate string `form:"dropoff_date" json:"dropoff_date"`

	Weight        float64 `form:"weight"         json:"weight"`
	Volume        float64 `form:"volume"         json:"volume"`
	EquipmentType string  `form:"equipment_type" json:"equipment_type"`

	Rate         float64 `form:"rate"          json:"rate"`
	RateCurrency string  `form:"rate_currency" json:"rate_currency"`

	Distance float64 `form:"distance" json:"distance"`
	Duration float64 `form:"duration" json:"duration"`
}

type LoadUpdateRequest = LoadStoreRequest
