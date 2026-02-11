module github.com/profe-ajedrez/transwarp/adapter/echoadapter

go 1.25.7

// nolint:gomoddirectives
replace github.com/profe-ajedrez/transwarp => ../../

require (
	github.com/labstack/echo/v5 v5.0.3
	github.com/profe-ajedrez/transwarp v0.0.0-00010101000000-000000000000
)

require golang.org/x/time v0.14.0 // indirect
