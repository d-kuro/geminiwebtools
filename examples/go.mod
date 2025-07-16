module examples

go 1.24.0

toolchain go1.24.3

replace github.com/d-kuro/geminiwebtools => ../

require github.com/d-kuro/geminiwebtools v0.0.0-00010101000000-000000000000

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
)
