package db

type UpdateDTO struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	MiddleName  *string `json:"middle_name"`
	DateOfBirth *string `json:"date_of_birth"`
}
