module github.com/iaconlabs/transwarp/examples/chi-basic

go 1.25.7

// Reemplazos para que el ejemplo use tu código local en lugar de GitHub
// replace github.com/iaconlabs/transwarp => ../../

// replace github.com/iaconlabs/transwarp/adapter/chiadapter => ../../adapter/chiadapter

require (
	github.com/iaconlabs/transwarp v0.0.12
	github.com/iaconlabs/transwarp/adapter/chiadapter v0.0.9
)

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-chi/chi/v5 v5.2.5 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
