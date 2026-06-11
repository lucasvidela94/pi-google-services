#!/usr/bin/env node
/**
 * uninstall.js — Removes pi-google-services binary and Pi MCP config.
 */

const fs = require("fs");
const path = require("path");
const os = require("os");

const PI_MCP_PATH = path.join(os.homedir(), ".pi", "agent", "mcp.json");
const BIN_DIR = path.join(os.homedir(), ".local", "bin");
const BIN_NAME = "pi-google-services";
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);
const CONFIG_DIR = path.join(os.homedir(), ".config", "pi-google-services");

function main() {
	console.log("\n🗑  pi-google-services uninstaller\n");

	// Remove binary
	if (fs.existsSync(BIN_PATH)) {
		fs.unlinkSync(BIN_PATH);
		console.log("  ✓ Removed binary:", BIN_PATH);
	} else {
		console.log("  - Binary not found");
	}

	// Remove config dir
	if (fs.existsSync(CONFIG_DIR)) {
		fs.rmSync(CONFIG_DIR, { recursive: true, force: true });
		console.log("  ✓ Removed config:", CONFIG_DIR);
	}

	// Remove MCP config entry
	if (fs.existsSync(PI_MCP_PATH)) {
		try {
			const config = JSON.parse(fs.readFileSync(PI_MCP_PATH, "utf-8"));
			if (config.mcpServers && config.mcpServers["google-services"]) {
				delete config.mcpServers["google-services"];
				fs.writeFileSync(PI_MCP_PATH, JSON.stringify(config, null, 2) + "\n");
				console.log("  ✓ Removed from Pi MCP config");
			} else {
				console.log("  - Not found in Pi MCP config");
			}
		} catch {
			console.log("  ⚠ Could not parse Pi MCP config (manual cleanup needed)");
		}
	}

	console.log(
		"\n  ✅ Done. Tokens and cached data remain in ~/.config/pi-google-services/ if you want to reinstall later.\n",
	);
}

main();
