package oauth

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/db/models"
	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/rs/zerolog/log"
)

type Service struct {
	db            *db.Database
	ProviderIndex providerIndex
}

func New(db *db.Database) Service {
	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_KEY"), os.Getenv("GOOGLE_SECRET"), "http://localhost:8089/auth/google/callback"),
	)

	pi := providerIndex{Providers: []string{"google"}, ProvidersMap: map[string]string{"google": "Google"}}

	return Service{db: db, ProviderIndex: pi}
}

func (s Service) Providers() providerIndex {
	return s.ProviderIndex
}

type providerIndex struct {
	Providers    []string
	ProvidersMap map[string]string
	Services     []models.Service
}

func (s Service) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(coremiddleware.UserIDCtxKey).(uuid.UUID)
	err := s.db.DeleteAccessToken(r.Context(), userID)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, "oauth error", http.StatusInternalServerError)
		return
	}

	gothic.Logout(w, r)
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (s Service) CallBackHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, "oauth error", http.StatusInternalServerError)
		return
	}

	u := models.User{
		ID:        uuid.New(),
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		Firstname: user.FirstName,
		Lastname:  user.LastName,
		Role:      coremodels.USER,
		CreatedAt: time.Now(),
	}

	token := models.AccessToken{
		UserID:     u.ID,
		ExternalID: user.UserID,
		Token:      user.AccessToken,
		ExpiresAt:  user.ExpiresAt,
	}

	err = s.db.CreateUserAndToken(r.Context(), u, token)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, "oauth error", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(token); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, "oauth error", http.StatusInternalServerError)
		return
	}
}

func (s Service) AuthHandler(res http.ResponseWriter, req *http.Request) {
	// try to get the user without re-authenticating
	if gothUser, err := gothic.CompleteUserAuth(res, req); err == nil {
		t, _ := template.New("foo").Parse(userTemplate)
		t.Execute(res, gothUser)
	} else {
		gothic.BeginAuthHandler(res, req)
	}
}

var indexTemplate = `
<p><a href="/services">Add Service</a></p>
<h1>List of Services</h1>
	<ul>
		{{range .Services}}
		<li>{{.Name}} - Prefix: {{.Prefix}} - Required Roles: {{.RequiredRoles}} - Costs: {{.Costs}}</li>
		{{end}}
	</ul>

<h2>Providers</h2>
{{range $key,$value:=.Providers}}
    <p><a href="/auth/{{$value}}">Log in with {{index $.ProvidersMap $value}}</a></p>
{{end}}`

var userTemplate = `
<p><a href="/logout/{{.Provider}}">logout</a></p>
<p>Name: {{.Name}} [{{.LastName}}, {{.FirstName}}]</p>
<p>Email: {{.Email}}</p>
<p>NickName: {{.NickName}}</p>
<p>Location: {{.Location}}</p>
<p>AvatarURL: {{.AvatarURL}} <img src="{{.AvatarURL}}"></p>
<p>Description: {{.Description}}</p>
<p>UserID: {{.UserID}}</p>
<p>AccessToken: {{.AccessToken}}</p>
<p>ExpiresAt: {{.ExpiresAt}}</p>
<p>RefreshToken: {{.RefreshToken}}</p>
`
