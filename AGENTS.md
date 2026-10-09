# pi-google-services

CLI + MCP server (stdio) en Go. `main.go` trae los comandos;
`internal/<dominio>` habla con cada API de Google; `internal/services`
registra las tools MCP; `internal/mcp` es el protocolo.

## Gate (correr a mano al terminar, siempre)

```sh
gofmt -l .                 # limpio = sin salida
go vet ./...
golangci-lint run ./...    # config en .golangci.yml, 0 issues o no avanza
go test -race ./...
```

opencode formatea con gofmt solo (`opencode.json`) y re-corre el gate
cuando la sesión queda idle (`.opencode/plugins/go-gate.js`), pero el
plugin puede fallar en silencio: el gate manual es obligatorio.

## Reglas

- Errores con contexto: `fmt.Errorf("...: %w", err)`.
- Todo HTTP, subproceso y llamada Google API con `ctx` (`NewRequestWithContext`,
  `CommandContext`, `.Context(ctx)`). En tests `context.TODO()`, nunca `nil`.
- Sin `panic`, sin `init()`, sin `util/` ni `common/`, sin globales nuevos.
- `http.Server` siempre con `ReadHeaderTimeout`.
- Tests table-driven, sin mocks hechos a medida para hacer pasar algo.
- Al agregar una tool: registrarla en `internal/services`, actualizar los
  tests de conteo y documentarla en `SKILL.md`.
