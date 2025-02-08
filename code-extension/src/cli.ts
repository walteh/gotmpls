/**
 * CLI Engine Implementation
 *
 * This implementation uses the gotmpls CLI tool to provide language server functionality.
 * It spawns a child process running the gotmpls serve-lsp command and communicates with it
 * using the Language Server Protocol.
 *
 * Process Architecture:
 * ```
 *  +----------------+         +-----------------+
 *  |   VS Code      |   LSP   |    gotmpls     |
 *  |  Extension     |<------->|    serve-lsp   |
 *  +----------------+         +-----------------+
 *                   (stdio transport)
 * ```
 */

import * as fs from "fs";
import * as path from "path";
import { exec, spawn } from "child_process";
import { promisify } from "util";

import * as vscode from "vscode";

import { IPCMessageReader, IPCMessageWriter, MessageTransports } from "vscode-languageclient/node";

import { BaseGotmplsEngine, getConfig, GotmplsEngineType } from "./engine";

const execAsync = promisify(exec);

export class CLIEngine extends BaseGotmplsEngine {
	private executable: string | undefined;
	private version: string | undefined;

	constructor() {
		super(GotmplsEngineType.CLI);
	}

	async initialize(context: vscode.ExtensionContext, outputChannel: vscode.OutputChannel): Promise<void> {
		this.outputChannel = outputChannel;
		this.log("Initializing CLI engine...");

		try {
			// Find the executable
			this.executable = await this.findExecutable();
			this.log(`Found executable: ${this.executable}`);

			// Get version
			this.version = await this.getVersion(context);
			this.log(`Version: ${this.version}`);

			this.initialized = true;
			this.log("CLI engine initialized");
		} catch (err) {
			this.log(`Error initializing CLI engine: ${err}`);
			throw err;
		}
	}

	async createTransport(context: vscode.ExtensionContext): Promise<MessageTransports> {
		if (!this.executable) {
			throw new Error("Executable not found");
		}

		const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
		if (!workspaceFolder) {
			throw new Error("No workspace folder found");
		}

		const config = getConfig();

		// Create a promise that resolves with the StreamInfo
		return new Promise<MessageTransports>((resolve, reject) => {
			const serverProcess = spawn(this.executable!, ["serve-lsp", config.debug ? "--debug" : ""], {
				cwd: workspaceFolder.uri.fsPath,
				env: {
					...process.env,
					GOTMPL_DEBUG: config.debug ? "1" : "0",
					GOPATH: process.env.GOPATH || path.join(process.env.HOME || "", "go"),
					GOMODCACHE: process.env.GOMODCACHE || path.join(process.env.HOME || "", "go", "pkg", "mod"),
					GO111MODULE: "on",
				},
			});

			serverProcess.stderr.on("data", (data) => {
				this.log(`[stderr] ${data}`);
			});

			serverProcess.on("error", (err) => {
				this.log(`Process error: ${err}`);
				reject(err);
			});

			serverProcess.on("exit", (code, signal) => {
				this.log(`Process exited with code ${code} and signal ${signal}`);
				if (code !== 0) {
					reject(new Error(`Process exited with code ${code}`));
				}
			});

			// Create LSP message reader and writer
			const reader = new IPCMessageReader(serverProcess);
			const writer = new IPCMessageWriter(serverProcess);

			// Return the transport
			resolve({
				reader,
				writer,
			});
		});
	}

	async getVersion(context: vscode.ExtensionContext): Promise<string> {
		if (this.version) {
			return this.version;
		}

		if (!this.executable) {
			throw new Error("Executable not found");
		}

		try {
			const { stdout } = await execAsync(`${this.executable} version`);
			this.version = stdout.trim();
			return this.version;
		} catch (err) {
			throw new Error(`Failed to get gotmpls version: ${err}`);
		}
	}

	private async findExecutable(): Promise<string> {
		const config = getConfig();
		let executable = config.executable || "gotmpls";

		// If it's an absolute path, verify it exists
		if (path.isAbsolute(executable)) {
			if (fs.existsSync(executable)) {
				return executable;
			}
			throw new Error(`Executable not found at configured path: ${executable}`);
		}

		// If we have a workspace folder, check relative to that
		const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
		if (workspaceFolder && !executable.startsWith("./") && !executable.startsWith("../")) {
			const workspacePath = path.join(workspaceFolder.uri.fsPath, executable);
			if (fs.existsSync(workspacePath)) {
				return workspacePath;
			}
		}

		// Check if it's in PATH
		const envPath = process.env.PATH || "";
		const pathSeparator = process.platform === "win32" ? ";" : ":";
		const pathDirs = envPath.split(pathSeparator);

		for (const dir of pathDirs) {
			const fullPath = path.join(dir, executable);
			if (fs.existsSync(fullPath)) {
				return fullPath;
			}
		}

		throw new Error(
			`Executable '${executable}' not found in PATH. Please install it with 'go install github.com/walteh/gotmpls/cmd/gotmpls@latest'`,
		);
	}
}
