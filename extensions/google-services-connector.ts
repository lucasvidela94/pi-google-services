// Auto-connects the google-services MCP server on session start.
// Zero friction: user installs the package, restarts Pi, tools are ready.
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
	pi.on("session_start", async (_event) => {
		// Queue MCP reconnect as a follow-up command
		pi.sendUserMessage("/mcp reconnect google-services", {
			deliverAs: "nextTurn",
		});
	});
}
