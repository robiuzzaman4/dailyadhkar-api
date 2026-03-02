## **Backend Server/API High Level Overview**

## **Prjoect Name: `github.com/robiuzzaman4/daily-durood-api`**

- DDD Pattern (Must)
- Load Env from `.env` With some validation
- Handle Database correctly & shutdown server gracefully
- No use any framework for http use `net-http`
- Handle concurrently email sending `EMAIL_SEND_LIMIT` from .env
- Send email at every day `EMAIL_SEND_TIME` am/pm;
- Design email template content will provide later.

### **User Model**

```go
type Role string

const (
    RoleUser  Role = "user"
    RoleAdmin Role = "admin"
)

type User struct {
    ID                  string         `json:"id"`      // comes from clerk
    Name                string         `json:"name"`    // comes from clerk
    Email               string         `json:"email"`   // comes from clerk
    IsSubscribed        bool           `json:"is_subscribed"` // default true
    TotalEmailReceived  int            `json:"total_email_received"`
    Role                Role           `json:"role"` // default user
}
```
### **Example Env Variables (Note: You need add/remove/update this list based on industry standard naming convention) and other required variables**

```.env
DATABASE_URL=
UNOSEND_API_KEY=
EMAIL_SEND_TIME=
EMAIL_SEND_LIMIT=
```

