module github.com/iaconlabs/transwarp/adapter/muxadapter

go 1.25.7

retract (   
    [v0.0.1, v0.0.25] // deprecated
)

// nolint:gomoddirectives
// replace github.com/iaconlabs/transwarp => ../../

require github.com/iaconlabs/transwarp v0.0.12
