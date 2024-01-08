# build with tinygo to use wasi wasm target, but do not include debug info, making binary much smaller
go build -ldflags "-s -w" .