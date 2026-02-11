module github.com/profe-ajedrez/transwarp/examples/echo-basic

go 1.25.7

// Reemplazos para desarrollo local
replace github.com/profe-ajedrez/transwarp => ../../

replace github.com/profe-ajedrez/transwarp/adapter/echoadapter => ../../adapter/echoadapter

require (
	github.com/profe-ajedrez/transwarp v0.0.0-00010101000000-000000000000
	github.com/profe-ajedrez/transwarp/adapter/echoadapter v0.0.0-00010101000000-000000000000
)

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/labstack/echo/v5 v5.0.3 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
