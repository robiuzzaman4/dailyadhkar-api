package user

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

type User struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	IsSubscribed       bool   `json:"is_subscribed"`
	TotalEmailReceived int    `json:"total_email_received"`
	Role               Role   `json:"role"`
	Gender             Gender `json:"gender"`
}
