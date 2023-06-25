package public

import (
	"net/http"
	"text/template"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/gorilla/mux"
)

type Service struct {
	db *db.Database
}

func New(db *db.Database) Service {
	return Service{
		db: db,
	}
}

func (s Service) Router(r *mux.Router) {
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		services, err := s.db.GetServices(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Define a template to render the list of services
		tmpl := `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Services</title>
			</head>
			<body>
				<h1>Services</h1>
				<ul>
					{{range .}}
					<li>{{.Name}}</li>
					{{end}}
				</ul>
			</body>
			</html>
		`

		// Create a template instance
		t, err := template.New("services").Parse(tmpl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Execute the template with the list of services
		err = t.Execute(w, services)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
