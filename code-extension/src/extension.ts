/**
 * Gotmpls VS Code Extension
 *
 * This extension provides language server capabilities for Go templates.
 * It supports multiple engine implementations (CLI, WASM, WASI) through a common interface.
 *
 * Architecture Overview:
 * ```
 *  +----------------+
 *  |   VS Code      |
 *  |   Extension    |
 *  +----------------+
 *          |
 *  +----------------+
 *  |  GotmplsEngine |
 *  +----------------+
 *          |
 *  +-------+--------+
 *  |               |
 *  v               v
 * CLI            WASM
 * ```
 */

import * as vscode from "vscode";

import { CLIEngine } from "./cli";
import { getConfig, GotmplsEngine, GotmplsEngineType } from "./engine";
import { WasiEngine } from "./wasi";
import { WasmEngine } from "./wasm";

// Current engine instance
let currentEngine: GotmplsEngine | undefined;

export async function activate(context: vscode.ExtensionContext) {
	const outputChannel = vscode.window.createOutputChannel("gotmpls");
	outputChannel.show();
	outputChannel.appendLine("🚀 gotmpls is now active");

	try {
		// Get configuration
		const config = getConfig();

		// Create engine instance based on configuration
		currentEngine = createEngine(config.engine, outputChannel);
		outputChannel.appendLine(`📦 Using ${config.engine} engine`);

		console.log("hi");

		// Initialize engine
		await currentEngine.initialize(context, outputChannel);
		outputChannel.appendLine("✅ Engine initialized");

		// Start server
		await currentEngine.startServer(context);
		outputChannel.appendLine("🌟 Language server started");

		// Register cleanup
		context.subscriptions.push({
			dispose: () => {
				currentEngine?.stopServer(context);
				outputChannel.dispose();
			},
		});
	} catch (err) {
		outputChannel.appendLine(`❌ Error activating extension: ${err}`);
		throw err;
	}
}

export function deactivate(context: vscode.ExtensionContext): Thenable<void> | undefined {
	if (!currentEngine) {
		return undefined;
	}
	return currentEngine.stopServer(context);
}

var wasiEngine: WasiEngine | undefined;
var wasmEngine: WasmEngine | undefined;

function createEngine(type: GotmplsEngineType, outputChannel: vscode.OutputChannel): GotmplsEngine {
	switch (type) {
		case GotmplsEngineType.CLI:
			return new CLIEngine();
		case GotmplsEngineType.WASM:
			if (!wasmEngine) {
				wasmEngine = new WasmEngine(outputChannel);
			}
			return wasmEngine;
		case GotmplsEngineType.WASI:
			if (!wasiEngine) {
				wasiEngine = new WasiEngine(outputChannel);
			}
			return wasiEngine;
		default:
			throw new Error(`Unknown engine type: ${type}`);
	}
}
