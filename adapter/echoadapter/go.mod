module github.com/iaconlabs/transwarp/adapter/echoadapter

go 1.25.7

// nolint:gomoddirectives
// replace github.com/iaconlabs/transwarp => ../../

require (
	github.com/iaconlabs/transwarp v0.0.12
	github.com/labstack/echo/v5 v5.0.3
)

require golang.org/x/time v0.14.0 // indirect
