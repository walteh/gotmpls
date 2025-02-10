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

import * as path from "path";
import { exec, spawn } from "child_process";
import { promisify } from "util";

import * as vscode from "vscode";

import { MessageTransports, StreamInfo, StreamMessageReader, StreamMessageWriter } from "vscode-languageclient/node";

import { BaseGotmplsEngine, getConfig } from "./engine";

const execAsync = promisify(exec);

// 🔧 Base class for CLI-based engines
abstract class BaseCliEngine extends BaseGotmplsEngine {
	protected version: string | undefined;

	abstract getCommand(): Promise<{ path: string; args: string[] }>;

	async initialize(
		context: vscode.ExtensionContext,
		outputChannel: vscode.OutputChannel,
	): Promise<MessageTransports> {
		this.outputChannel = outputChannel;
		this.log("Initializing CLI engine...");

		try {
			// Get version
			this.version = await this.getVersion(context);
			this.log(`Version: ${this.version}`);

			this.initialized = true;
			this.log("CLI engine initialized");
		} catch (err) {
			this.log(`Error initializing CLI engine: ${err}`);
			throw err;
		}

		return this.createTransport(context);
	}

	async createTransport(context: vscode.ExtensionContext): Promise<MessageTransports> {
		const { path: cmd, args } = await this.getCommand();
		const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
		if (!workspaceFolder) {
			throw new Error("No workspace folder found");
		}

		const config = getConfig();

		return new Promise<MessageTransports>((resolve, reject) => {
			this.log(`Spawning ${cmd} ${args.join(" ")}`);
			const serverProcess = spawn(cmd, [...args, "serve-lsp", config.debug ? "--debug" : ""], {
				cwd: workspaceFolder.uri.fsPath,
				env: {
					...process.env,
					GOTMPL_DEBUG: config.debug ? "1" : "0",
					GOPATH: process.env.GOPATH || path.join(process.env.HOME || "", "go"),
					GOMODCACHE: process.env.GOMODCACHE || path.join(process.env.HOME || "", "go", "pkg", "mod"),
					GO111MODULE: "on",
				},
				stdio: ["pipe", "pipe", "pipe"], // Explicitly set up pipes
			});

			// Debug logging for process streams
			this.log("Server process created");

			serverProcess.stdout.on("data", (data) => {
				this.log(`[stdout] ${data.toString()}`);
			});

			serverProcess.stderr.on("data", (data) => {
				this.log(`[stderr] ${data.toString()}`);
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

			// Create a StreamInfo object
			const streamInfo: StreamInfo = {
				writer: serverProcess.stdin,
				reader: serverProcess.stdout,
				detached: false,
			};

			// Debug logging for raw streams
			serverProcess.stdout.on("data", (data) => {
				this.log(`[Raw stdout] ${data.toString()}`);
			});

			serverProcess.stdin.on("error", (error) => {
				this.log(`[stdin error] ${error}`);
			});

			// wait for the server to be ready
			setTimeout(() => {
				resolve({
					reader: new StreamMessageReader(streamInfo.reader),
					writer: new StreamMessageWriter(streamInfo.writer),
					detached: false,
				});
			}, 1000);
		});
	}

	async getVersion(context: vscode.ExtensionContext): Promise<string> {
		if (this.version) {
			return this.version;
		}

		const { path: cmd, args } = await this.getCommand();
		try {
			const { stdout } = await execAsync(`${cmd} ${args.join(" ")} raw-version`);
			this.version = stdout.trim();
			return this.version;
		} catch (err) {
			throw new Error(`Failed to get gotmpls version: ${err}`);
		}
	}
}
