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
import { WasmEngine } from "./wasm";

// Current engine instance
// let currentEngine: GotmplsEngine | undefined;

export async function activate(context: vscode.ExtensionContext) {
	const outputChannel = vscode.window.createOutputChannel("gotmpls");
	outputChannel.show();
	outputChannel.appendLine("🚀 gotmpls is now active");

	try {
		// Get configuration
		const config = getConfig();

		// Create engine instance based on configuration
		outputChannel.appendLine(`📦 Using ${config.engine} engine`);

		const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
		if (!workspaceFolder) {
			throw new Error("No workspace folder found");
		}

		// const baseClientOptions = createClientOptions(workspaceFolder, outputChannel);

		// if (config.engine === GotmplsEngineType.WASI) {
		// 	await wasi_activate(context, outputChannel, baseClientOptions);
		// 	outputChannel.appendLine("🌟 WASI engine started");
		// } else {
		const engine = createEngine(config.engine, outputChannel);
		// Initialize engine
		const messageTransports = engine.initialize(context, outputChannel);
		outputChannel.appendLine("✅ Engine initialized");

		// Start server
		await engine.startServer(context, messageTransports);
		outputChannel.appendLine("🌟 Language server started");
		context.subscriptions.push({
			dispose: () => {
				engine.stopServer(context);
				outputChannel.dispose();
			},
		});
		// }
		// Register cleanup
	} catch (err) {
		outputChannel.appendLine(`❌ Error activating extension: ${err}`);
		throw err;
	}
}

// export function deactivate(context: vscode.ExtensionContext): Thenable<void> | undefined {
// 	if (!currentEngine) {
// 		return undefined;
// 	}
// 	return currentEngine.stopServer(context);
// }

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
		default:
			throw new Error(`Unknown engine type: ${type}`);
	}
}
