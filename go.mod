module github.com/MohamedElashri/nida

go 1.26.0

require (
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/dop251/goja v0.0.0-20261019115134-f7f3edb84bee
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/tdewolff/minify/v2 v2.24.21
	github.com/yuin/goldmark v1.8.6
	go.yaml.in/yaml/v3 v3.0.5
	golang.org/x/image v0.46.0
)

require (
	github.com/dlclark/regexp2/v2 v2.8.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/tdewolff/parse/v2 v2.8.18 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace github.com/dop251/goja => ./third_party/goja
