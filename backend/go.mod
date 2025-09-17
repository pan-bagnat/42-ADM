module adm-backend

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.11
	github.com/go-chi/cors v1.2.1
	github.com/lib/pq v1.10.9
	github.com/oklog/ulid/v2 v2.1.1
)

replace github.com/go-chi/chi/v5 => ./vendor/github.com/go-chi/chi/v5
replace github.com/go-chi/cors => ./vendor/github.com/go-chi/cors
replace github.com/lib/pq => ./vendor/github.com/lib/pq
replace github.com/oklog/ulid/v2 => ./vendor/github.com/oklog/ulid/v2
