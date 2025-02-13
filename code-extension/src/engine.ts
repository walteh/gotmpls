/**
 * Gotmpls Engine Interface
 *
 * This file defines the core interfaces and types for the Gotmpls language server.
 * The architecture supports multiple engine implementations (CLI, WASM) through a common interface.
 *
 * Architecture:
 * ```
 *                     +----------------+
 *                     |  GotmplsEngine |
 *                     +----------------+
 *                            ^
 *                            |
 *             +-------------+-------------+
 *             |                           |
 *     +--------------+           +--------------+
 *     |  WasmEngine  |           |   CLIEngine  |
 *     +--------------+           +--------------+
 * ```
 */

import * as vscode from "vscode";

import { LanguageClient, LanguageClientOptions, MessageTransports } from "vscode-languageclient/node";

// 🔧 Engine type enum
export enum GotmplsEngineType {
	WASM = "wasm",
	WASI = "wasi",
	BUNDLED = "bundled",
	GO_TOOL = "go-tool",
	GO_RUN = "go-run",
	SYSTEM_PATH = "system-path",
	LOCAL = "local",
	CLI = "cli", // kept for backward compatibility
}

// 📝 Language IDs enum
export enum VSCodeLanguageID {
	GOTMPL = "gotmpl",
	GO = "go",
}

// 🎯 Supported languages array
export const SUPPORTED_LANGUAGES: VSCodeLanguageID[] = [VSCodeLanguageID.GOTMPL, VSCodeLanguageID.GO];

// 🔌 Interface for engine implementations
export interface GotmplsEngine {
	/**
	 * Initialize the engine with the given extension context
	 * @param context VS Code extension context
	 * @param outputChannel Output channel for logging
	 */
	initialize(context: vscode.ExtensionContext, outputChannel: vscode.OutputChannel): Promise<MessageTransports>;

	/**
	 * Start the language server
	 * @returns LanguageClient instance
	 */
	startServer(
		context: vscode.ExtensionContext,
		messageTransports: Promise<MessageTransports>,
	): Promise<LanguageClient>;

	/**
	 * Stop the language server
	 */
	stopServer(context: vscode.ExtensionContext): Promise<void>;

	/**
	 * Get the engine version
	 */
	getVersion(context: vscode.ExtensionContext): Promise<string>;

	/**
	 * Check if the engine is initialized
	 */
	isInitialized(): boolean;

	/**
	 * Get the engine type
	 */
	getType(): GotmplsEngineType;
}

export function createClientOptions(
	workspaceFolder: vscode.WorkspaceFolder,
	outputChannel: vscode.OutputChannel,
): LanguageClientOptions {
	const config = getConfig();
	return {
		documentSelector: [{ scheme: "file", language: "gotmpl" }],
		synchronize: {
			fileEvents: vscode.workspace.createFileSystemWatcher("**/*.{tmpl,go}"),
			configurationSection: "gotmpls",
		},
		workspaceFolder: workspaceFolder,
		outputChannel: outputChannel,
		traceOutputChannel: outputChannel,
		initializationOptions: {
			trace: {
				server: config.trace.server,
			},
		},
	};
}

// 🛠️ Base class for engine implementations
export abstract class BaseGotmplsEngine implements GotmplsEngine {
	protected initialized: boolean = false;
	protected client: LanguageClient | undefined;
	protected outputChannel: vscode.OutputChannel | undefined;

	constructor(protected engineType: GotmplsEngineType) {
		this.client = undefined;
		this.initialized = false;
		this.outputChannel = undefined;
	}

	abstract initialize(
		context: vscode.ExtensionContext,
		outputChannel: vscode.OutputChannel,
	): Promise<MessageTransports>;
	abstract getVersion(context: vscode.ExtensionContext): Promise<string>;

	/**
	 * Creates common client options for LSP
	 */
	protected createClientOptions(workspaceFolder: vscode.WorkspaceFolder): LanguageClientOptions {
		const config = getConfig();
		return {
			documentSelector: [{ scheme: "file", language: "gotmpl" }],
			synchronize: {
				fileEvents: vscode.workspace.createFileSystemWatcher("**/*.{tmpl,go}"),
				configurationSection: "gotmpls",
			},
			workspaceFolder: workspaceFolder,
			outputChannel: this.outputChannel,
			traceOutputChannel: this.outputChannel,
			initializationOptions: {
				trace: {
					server: config.trace.server,
				},
			},
		};
	}

	async startServer(
		context: vscode.ExtensionContext,
		messageTransports: Promise<MessageTransports>,
	): Promise<LanguageClient> {
		const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
		if (!workspaceFolder) {
			throw new Error("No workspace folder found");
		}

		// Create the language client
		this.client = new LanguageClient(
			"gotmpls",
			"gotmpls",
			() => messageTransports,
			this.createClientOptions(workspaceFolder),
		);

		// Set up telemetry handling
		this.setupTelemetry();

		await this.client.start();
		return this.client;
	}

	async stopServer(context: vscode.ExtensionContext): Promise<void> {
		if (this.client) {
			await this.client.stop();
			this.client = undefined;
		}
	}

	isInitialized(): boolean {
		return this.initialized;
	}

	getType(): GotmplsEngineType {
		return this.engineType;
	}

	protected log(message: string): void {
		this.outputChannel?.appendLine(`[${this.engineType}] ${message}`);
	}

	private setupTelemetry(): void {
		if (!this.client) return;

		const config = getConfig();

		this.client.onNotification("telemetry/event", (params: any) => {
			var str = "";
			switch (params.type) {
				case 1:
					str = `🟥 error      `;
					break;
				case 2:
					str = `🟧 warning    `;
					break;
				case 3:
					str = `🟦 info       `;
					break;
				case 5:
					str = `🟪 debug      `;
					break;
				case 4:
					str = `⬜ trace      `;
					break;
				case 6:
					str = `⬜ dependency `;
					break;
			}

			// if trace is disabled, skip 4 and 6
			if (!config.trace.server) {
				if (params.type === 4 || params.type === 6) return;
			}

			// Add time and source if available
			if (params.time) str += `${params.time} `;
			if (params.source) str += `${params.source} `;

			// Add message
			str += `- ${params.message}`;

			// Add extra fields if any
			if (params.extra) {
				const extras = Object.entries(params.extra)
					.map(([key, value]) => `${key}=${value}`)
					.join(" ");
				if (extras) str += ` | ${extras}`;
			}

			this.log(str);
		});
	}
}

// 🎛️ Configuration interface
export interface GotmplsConfig {
	engine: GotmplsEngineType;
	executable?: string;
	debug: boolean;
	trace: {
		server: boolean;
	};
}

// 📦 Helper to get configuration
export function getConfig(): GotmplsConfig {
	const config = vscode.workspace.getConfiguration("gotmpls");
	return {
		engine: config.get<GotmplsEngineType>("engine") || GotmplsEngineType.WASM,
		executable: config.get<string>("executable"),
		debug: config.get<boolean>("debug") || false,
		trace: {
			server: config.get<boolean>("trace.server") || false,
		},
	};
}
