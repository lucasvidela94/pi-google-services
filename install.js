#!/usr/bin/env node
/**
 * install.js — ensure pi-google-services is installed and wired into Pi.
 *
 * Idempotent. Safe to run more than once. Used both as npm postinstall and by
 * the Pi extension to self-heal when npm's allowScripts blocked the postinstall.
 *
 * The npm tarball already ships the platform binaries, so by default nothing
 * is downloaded: assets are unpacked from the local package.
 * GitHub Releases is only a fallback for development checkouts without bin/.
 *
 * OAuth note: the binary ships with a pre-registered public OAuth client ID
 * baked in at release time (see internal/auth/embedded.go). No credentials
 * are downloaded or copied at install — your tokens stay on your machine.
 */

const fs = require("fs");
const os = require("os");
const path = require("path");
const zlib = require("zlib");
const { execSync } = require("child_process");

const PKG = require("./package.json");
const REPO = PKG.repository.url.replace("git+", "").replace(".git", "");
const VERSION = "v" + PKG.version;

const PKG_DIR = __dirname;
const PI_AGENT_DIR = path.join(os.homedir(), ".pi", "agent");
const PI_MCP_PATH = path.join(PI_AGENT_DIR, "mcp.json");
const BIN_DIR = path.join(os.homedir(), ".local", "bin");
const BIN_NAME = "pi-google-services";
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);

function platform() {
	const arch = os.arch();
	const plat = os.platform();

	if (plat === "linux" && arch === "x64") return "linux-amd64";
	if (plat === "linux" && arch === "arm64") return "linux-arm64";
	if (plat === "darwin" && arch === "x64") return "darwin-amd64";
	if (plat === "darwin" && arch === "arm64") return "darwin-arm64";

	console.error(`Unsupported platform: ${plat}-${arch}`);
	console.error("Supported: linux-x64, linux-arm64, darwin-x64, darwin-arm64");
	process.exit(1);
}

function download(url, dest) {
	return new Promise((resolve, reject) => {
		const https = require("https");
		const file = fs.createWriteStream(dest);
		https
			.get(url, (res) => {
				if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
					file.close();
					fs.unlinkSync(dest);
					return download(res.headers.location, dest).then(resolve).catch(reject);
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

function localAsset(...segments) {
	return path.join(PKG_DIR, ...segments);
}

function binaryInstalled() {
	if (!fs.existsSync(BIN_PATH)) return false;
	try {
		const out = execSync(
			`${BIN_PATH} --version 2>/dev/null || ${BIN_PATH} help 2>&1 || true`,
			{ encoding: "utf-8" },
		);
		return out.includes("v" + VERSION) || out.includes(VERSION);
	} catch {
		return false;
	}
}

async function ensureBinary() {
	if (binaryInstalled()) {
		console.log(`  ✓ Binary already up-to-date (v${VERSION})`);
		return;
	}

	const assetName = `pi-google-services-${platform()}.gz`;
	const localSrc = localAsset("bin", assetName);

	let compressed;
	if (fs.existsSync(localSrc)) {
		console.log(`  ✓ Using bundled binary ${assetName}`);
		compressed = fs.readFileSync(localSrc);
	} else {
		const url = `${REPO}/releases/download/${VERSION}/${assetName}`;
		const dest = path.join(os.tmpdir(), assetName);
		console.log(`  ⬇ Downloading ${assetName}...`);
		try {
			await download(url, dest);
			compressed = fs.readFileSync(dest);
			fs.unlinkSync(dest);
		} catch (err) {
			console.error(`  ❌ Binary download failed: ${err.message}`);
			console.error(
				"     Build from source instead: git clone ... && go build -o pi-google-services .",
			);
			process.exit(1);
		}
	}

	fs.mkdirSync(BIN_DIR, { recursive: true });
	const binary = zlib.gunzipSync(compressed);
	// Write to a temp file and rename over the target. On Linux you cannot
	// overwrite an executable that is currently running (ETXTBSY: text file
	// busy), but rename atomically swaps the directory entry, so the running
	// process keeps its old inode and the next launch picks up the new binary.
	const tmpPath = `${BIN_PATH}.tmp-${process.pid}`;
	fs.writeFileSync(tmpPath, binary, { mode: 0o755 });
	try {
		fs.renameSync(tmpPath, BIN_PATH);
	} catch (err) {
		try {
			fs.unlinkSync(tmpPath);
		} catch {}
		throw err;
	}
	console.log(`  ✓ Installed to ${BIN_PATH}`);
}

function ensureMcpConfig() {
	let config;
	if (fs.existsSync(PI_MCP_PATH)) {
		try {
			config = JSON.parse(fs.readFileSync(PI_MCP_PATH, "utf-8"));
		} catch (err) {
			console.error(`  ❌ ${PI_MCP_PATH} is not valid JSON.`);
			console.error("     Refusing to overwrite it. Fix it, then re-run install.js.");
			console.error(`     Parse error: ${err.message}`);
			return false;
		}
	} else {
		config = { mcpServers: {} };
	}

	if (!config.mcpServers) config.mcpServers = {};
	if (config.mcpServers["google-services"]) {
		console.log("  ✓ google-services already configured in Pi MCP");
		return true;
	}

	config.mcpServers["google-services"] = {
		command: BIN_PATH,
		args: ["serve"],
	};

	fs.mkdirSync(PI_AGENT_DIR, { recursive: true });
	fs.writeFileSync(PI_MCP_PATH, JSON.stringify(config, null, 2) + "\n");
	console.log("  ✓ Pi MCP config updated");
	return true;
}

async function main() {
	console.log("\n📦 pi-google-services installer");
	console.log("==============================\n");

	await ensureBinary();
	const mcpOk = ensureMcpConfig();

	if (!mcpOk) {
		console.error("\n  ❌ Installation incomplete: MCP config could not be resolved.");
		process.exit(1);
	}

	console.log("");
	console.log("  ─────────────────────────────────────");
	console.log("  ✅ pi-google-services installed!");
	console.log("");
	console.log("  Next step:");
	console.log("    pi-google-services setup");
	console.log("    (opens browser → authorize → ready)");
	console.log("");
	console.log("  After that, restart Pi and ask:");
	console.log('    "show my events" or "read my inbox"');
	console.log("  ─────────────────────────────────────\n");
}

main().catch((err) => {
	console.error("Install failed:", err.message);
	process.exit(1);
});
