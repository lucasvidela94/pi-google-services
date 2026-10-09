// GoGate: corre el gate de Go cuando la sesión queda idle y hubo edits.
// Si algo falla, le devuelve el output al agente para que lo arregle.
// Nunca rompe la sesión: todo error del plugin se ignora en silencio.
//
// El gate real está en .golangci.yml + AGENTS.md; esto es solo el cable.
import { execFileSync } from "node:child_process";

const GATE = "gofmt -l . && go vet ./... && golangci-lint run ./... && go test ./...";
const MAX_OUT = 4000;

export const GoGate = async ({ client, directory }) => {
  let dirty = false;
  let running = false;
  let lastSent = "";

  const runGate = () => {
    try {
      execFileSync("sh", ["-c", GATE], {
        cwd: directory,
        encoding: "utf8",
        timeout: 180000,
        stdio: ["ignore", "pipe", "pipe"],
      });
      return "";
    } catch (e) {
      const out = [e.stdout, e.stderr].filter(Boolean).join("\n");
      return (out || String(e)).slice(0, MAX_OUT);
    }
  };

  return {
    "tool.execute.after": async (input) => {
      try {
        if (input.tool !== "edit" && input.tool !== "write") return;
        const args = { ...(input.args || {}), ...(input.output?.args || {}) };
        const file = args.filePath || args.path || "";
        if (typeof file === "string" && file.endsWith(".go")) dirty = true;
      } catch {
        // nunca bloquear al agente por un error del plugin
      }
    },

    event: async ({ event }) => {
      try {
        if (!event || event.type !== "session.idle" || !dirty || running) return;
        running = true;
        try {
          const failures = runGate();
          dirty = false;
          // Anti-loop: si el fallo es idéntico al ya reportado, no insistir (lo agarra CI).
          if (failures && failures !== lastSent) {
            lastSent = failures;
            const sessionID = event.properties?.sessionID || event.properties?.sessionId;
            if (sessionID) {
              await client.session.prompt({
                path: { id: sessionID },
                body: {
                  parts: [
                    {
                      type: "text",
                      text: `Go gate falló. Arreglalo antes de seguir:\n\n${failures}`,
                    },
                  ],
                },
              });
            }
          }
          if (!failures) lastSent = "";
        } finally {
          running = false;
        }
      } catch {
        // nunca bloquear al agente por un error del plugin
      }
    },
  };
};
