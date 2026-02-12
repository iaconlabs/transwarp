module github.com/iaconlabs/transwarp/examples/custom-validation

go 1.25.7

// replace github.com/iaconlabs/transwarp => ../../

// replace github.com/iaconlabs/transwarp/adapter/muxadapter => ../../adapter/muxadapter

require (
	github.com/go-playground/validator/v10 v10.30.1
	github.com/iaconlabs/transwarp v0.0.9
	github.com/iaconlabs/transwarp/adapter/muxadapter v0.0.9
)

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
