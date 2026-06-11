#!/usr/bin/env node
/**
 * install.js — pi-google-services postinstall script.
 *
 * Downloads the correct binary for your platform from GitHub Releases,
 * installs it to ~/.local/bin/, and configures Pi's MCP.
 *
 * Run manually:  node install.js
 */

const https = require("https");
const fs = require("fs");
const path = require("path");
const os = require("os");
const zlib = require("zlib");

const PKG = require("./package.json");
const REPO = PKG.repository.url.replace("git+", "").replace(".git", "");
const VERSION = "v" + PKG.version;

const PI_MCP_PATH = path.join(os.homedir(), ".pi", "agent", "mcp.json");
const BIN_DIR = path.join(os.homedir(), ".local", "bin");
const BIN_NAME = "pi-google-services";
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);
const CONFIG_DIR = path.join(os.homedir(), ".config", "pi-google-services");

function platform() {
	const arch = os.arch();
	const plat = os.platform();

	if (plat === "linux" && arch === "x64") return "linux-x64";
	if (plat === "linux" && arch === "arm64") return "linux-arm64";
	if (plat === "darwin" && arch === "x64") return "darwin-x64";
	if (plat === "darwin" && arch === "arm64") return "darwin-arm64";

	console.error(`Unsupported platform: ${plat}-${arch}`);
	console.error("Supported: linux-x64, linux-arm64, darwin-x64, darwin-arm64");
	process.exit(1);
}

function download(url, dest) {
	return new Promise((resolve, reject) => {
		const file = fs.createWriteStream(dest);
		https
			.get(url, (res) => {
				if (res.statusCode >= 300 && res.location) {
					// Follow redirect
					file.close();
					fs.unlinkSync(dest);
					return download(res.location, dest).then(resolve).catch(reject);
				}
				if (res.statusCode !== 200) {
					file.close();
					fs.unlinkSync(dest);
					reject(new Error(`HTTP ${res.statusCode}: ${url}`));
					return;
				}
				res.pipe(file);
				file.on("finish", () => {
					file.close();
					resolve();
				});
			})
			.on("error", (err) => {
				file.close();
				fs.unlinkSync(dest, () => {});
				reject(err);
			});
	});
}

function setupMcpConfig() {
	if (!fs.existsSync(PI_MCP_PATH)) {
		console.log("  ⚠ Pi MCP config not found at", PI_MCP_PATH);
		console.log("  Skipping MCP auto-config. Add manually:");
		console.log(
			`    { "mcpServers": { "google-services": { "command": "${BIN_PATH}", "args": ["serve"] } } }`,
		);
		return;
	}

	let config;
	try {
		config = JSON.parse(fs.readFileSync(PI_MCP_PATH, "utf-8"));
	} catch {
		config = { mcpServers: {} };
	}

	if (!config.mcpServers) config.mcpServers = {};
	if (config.mcpServers["google-services"]) {
		console.log("  ✓ google-services already configured in Pi MCP");
		return;
	}

	config.mcpServers["google-services"] = {
		command: BIN_PATH,
		args: ["serve"],
	};

	fs.writeFileSync(PI_MCP_PATH, JSON.stringify(config, null, 2) + "\n");
	console.log("  ✓ Pi MCP config updated");
}

async function main() {
	console.log("\n📦 pi-google-services installer");
	console.log("==============================\n");

	const plat = platform();
	const assetName = `pi-google-services-${plat}.gz`;
	const url = `https://github.com/${REPO}/releases/download/${VERSION}/${assetName}`;
	const dest = path.join(os.tmpdir(), assetName);
	const binDir = BIN_DIR;

	// Ensure bin dir
	fs.mkdirSync(binDir, { recursive: true });

	// Download binary
	console.log(`  ⬇ Downloading ${assetName}...`);
	console.log(`  From: ${url}`);

	try {
		await download(url, dest);
	} catch (err) {
		console.error(`\n  ❌ Download failed: ${err.message}`);
		console.error("\n  Possible reasons:");
		console.error(`    • Release ${VERSION} not published yet`);
		console.error("    • Network issue");
		console.error("\n  Build from source instead:");
		console.error(
			"    git clone https://github.com/timolabs/pi-google-services.git",
		);
		console.error(
			"    cd pi-google-services && go build -o pi-google-services .",
		);
		console.error("    cp pi-google-services ~/.local/bin/");
		process.exit(1);
	}

	// Decompress
	const compressed = fs.readFileSync(dest);
	const binary = zlib.gunzipSync(compressed);
	fs.writeFileSync(BIN_PATH, binary, { mode: 0o755 });
	fs.unlinkSync(dest);

	console.log(`  ✓ Installed to ${BIN_PATH}`);

	// Create config dir
	fs.mkdirSync(CONFIG_DIR, { recursive: true });

	// Setup MCP
	setupMcpConfig();

	// Done
	console.log("\n  ─────────────────────────────────────");
	console.log("  ✅ pi-google-services installed!");
	console.log("");
	console.log("  Next steps:");
	console.log("    1. Run:  pi-google-services login");
	console.log("       (opens browser → authorize with Google)");
	console.log("    2. Restart Pi session");
	console.log("    3. Ask Pi to manage your calendar & email");
	console.log("  ─────────────────────────────────────\n");
}

main().catch((err) => {
	console.error("Install failed:", err.message);
	process.exit(1);
});
