/**
 * Gotmpls VS Code Extension
 *
 * This extension provides language server capabilities for Go templates.
 * It supports multiple engine implementations (CLI, WASM) through a common interface.
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
import { WasmEngine } from "./wasm";
import { GotmplsEngine, GotmplsEngineType, getConfig } from "./engine";

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
		currentEngine = createEngine(config.engine);
		outputChannel.appendLine(`📦 Using ${config.engine} engine`);

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

const outputChannel = vscode.window.createOutputChannel("gotmpls");
const wasmEngine = new WasmEngine(outputChannel);

function createEngine(type: GotmplsEngineType): GotmplsEngine {
	switch (type) {
		case GotmplsEngineType.CLI:
			return new CLIEngine();
		case GotmplsEngineType.WASM:
			return wasmEngine;
		default:
			throw new Error(`Unknown engine type: ${type}`);
	}
}
