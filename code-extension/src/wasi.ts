/**
 * WASI Engine Implementation
 *
 * This implementation uses WASI to run the language server directly in VS Code.
 * The WASI binary provides a standard system interface that we can use.
 *
 * Architecture:
 * ```
 *  +----------------+         +-----------------+
 *  |   VS Code      |         |    gotmpls     |
 *  |  Extension     |<------->|     WASI       |
 *  +----------------+         +-----------------+
 *                   (stdio transport)
 * ```
 */

import * as path from "path";

import * as vscode from "vscode";

import { Wasm, WasmPseudoterminal } from "@vscode/wasm-wasi/v1";
import {
	DataCallback,
	Disposable,
	Event,
	Message,
	MessageReader,
	MessageTransports,
	MessageWriter,
	PartialMessageInfo,
} from "vscode-languageclient/node";

import { BaseGotmplsEngine, GotmplsEngineType } from "./engine";

// WASI module interface
declare global {
	interface WasiEnv {
		args: string[];
		env: { [key: string]: string };
		preopens: { [key: string]: string };
	}
}

/**
 * Custom message reader for WASI communication
 */
class WasiMessageReader implements MessageReader {
	private readonly emitter = new vscode.EventEmitter<Message>();
	private pty: WasmPseudoterminal | undefined;

	constructor() {
		// Initialize
	}

	public setPty(pty: WasmPseudoterminal) {
		this.pty = pty;
		// Handle both JSON and non-JSON data
		let buffer = "";
		let contentLength: number | undefined;

		pty.onDidWrite((data: string) => {
			console.log("Raw stdout (debug):", data);

			// Append to buffer
			buffer += data;

			// Process complete messages
			while (true) {
				// If we don't have a content length yet, try to parse headers
				if (contentLength === undefined) {
					const headerMatch = buffer.match(/Content-Length: (\d+)\r\n\r\n/);
					if (!headerMatch) break;

					contentLength = parseInt(headerMatch[1], 10);
					buffer = buffer.substring(headerMatch[0].length);
				}

				// If we have a content length, try to read a message
				if (contentLength !== undefined && buffer.length >= contentLength) {
					const message = buffer.substring(0, contentLength);
					buffer = buffer.substring(contentLength);
					contentLength = undefined;

					try {
						const parsed = JSON.parse(message);
						console.log("Parsed LSP message:", parsed);
						this.emitter.fire(parsed);
					} catch (err) {
						console.error("Failed to parse LSP message:", err);
					}
				} else {
					break;
				}
			}
		});
	}

	public get onError(): Event<Error> {
		return new vscode.EventEmitter<Error>().event;
	}

	public get onClose(): Event<void> {
		return new vscode.EventEmitter<void>().event;
	}

	public get onPartialMessage(): Event<PartialMessageInfo> {
		return new vscode.EventEmitter<PartialMessageInfo>().event;
	}

	public listen(callback: DataCallback): Disposable {
		return this.emitter.event(callback);
	}

	public dispose(): void {
		this.emitter.dispose();
	}
}

/**
 * Custom message writer for WASI communication
 */
class WasiMessageWriter implements MessageWriter {
	private readonly errorEmitter = new vscode.EventEmitter<[Error, Message | undefined, number | undefined]>();
	private readonly closeEmitter = new vscode.EventEmitter<void>();
	private pty: WasmPseudoterminal | undefined;

	constructor() {
		// Initialize
	}

	public setPty(pty: WasmPseudoterminal) {
		this.pty = pty;
	}

	public get onError(): Event<[Error, Message | undefined, number | undefined]> {
		return this.errorEmitter.event;
	}

	public get onClose(): Event<void> {
		return this.closeEmitter.event;
	}

	public async write(msg: Message): Promise<void> {
		if (!this.pty) {
			return Promise.reject(new Error("No PTY available"));
		}

		try {
			// Format as LSP message with Content-Length header
			const msgStr = JSON.stringify(msg);
			const data = `Content-Length: ${Buffer.byteLength(msgStr, "utf8")}\r\n\r\n${msgStr}`;
			console.log("Raw stdin (formatted LSP):", data);
			this.pty.write(data);
			return Promise.resolve();
		} catch (err) {
			const error = err instanceof Error ? err : new Error(String(err));
			this.errorEmitter.fire([error, msg, undefined]);
			return Promise.reject(error);
		}
	}

	public end(): void {
		this.closeEmitter.fire();
	}

	public dispose(): void {
		this.errorEmitter.dispose();
		this.closeEmitter.dispose();
	}
}

export class WasiEngine extends BaseGotmplsEngine {
	private wasi: Wasm | null = null;
	private reader: WasiMessageReader;
	private writer: WasiMessageWriter;
	private inputPty: WasmPseudoterminal | undefined;
	private outputPty: WasmPseudoterminal | undefined;
	private debugPty: WasmPseudoterminal | undefined; // Keep PTY for debug output

	constructor(outputChannel: vscode.OutputChannel) {
		super(GotmplsEngineType.WASI);
		this.outputChannel = outputChannel;

		// Create reader and writer
		this.reader = new WasiMessageReader();
		this.writer = new WasiMessageWriter();
	}

	async createTransport(context: vscode.ExtensionContext): Promise<MessageTransports> {
		return {
			reader: this.reader,
			writer: this.writer,
		};
	}

	async initialize(context: vscode.ExtensionContext): Promise<void> {
		if (this.initialized) {
			return;
		}

		this.log("🔍 Starting WASI initialization...");
		try {
			// Create WASI instance
			this.log("📦 Creating WASI instance...");
			const wasm = await Wasm.load();
			this.wasi = wasm;
			this.log("✅ WASI instance created successfully");

			// Create WASI process with PTY stdio
			this.log("🏗️  Creating PTYs...");

			// Create separate PTYs for LSP communication and debug
			this.inputPty = wasm.createPseudoterminal(); // For client -> server
			this.outputPty = wasm.createPseudoterminal(); // For server -> client
			this.debugPty = wasm.createPseudoterminal(); // For debug output
			this.log("✅ Created PTYs for LSP communication and debug");

			// Create a terminal for debug output
			this.log("🚀 Creating VS Code debug terminal...");
			const debugTerminal = vscode.window.createTerminal({
				name: "gotmpls LSP Debug",
				pty: this.debugPty as unknown as vscode.Pseudoterminal,
			});

			// Add verbose debug output handler
			this.debugPty.onDidWrite((data: string) => {
				// Log raw data with special characters visible
				console.log("Debug raw data:", JSON.stringify(data));
				// Log to extension output
				this.log(`[Debug] ${data.trim()}`);
				// Write to debug terminal
				debugTerminal.sendText(data, false);
			});

			// Load the WASI module
			const wasmPath = path.join(context.extensionPath, "out", "gotmpls.wasi.wasm");
			this.log(`📂 Loading WASI module from: ${wasmPath}`);
			const wasmBits = await vscode.workspace.fs.readFile(vscode.Uri.file(wasmPath));
			this.log(`📥 WASI module loaded, size: ${wasmBits.length} bytes`);

			this.log("🔧 Compiling WASI module...");
			const module = await WebAssembly.compile(wasmBits);
			this.log("✅ WASI module compiled successfully");

			// Create WASI process with split stdio
			const process = await this.wasi.createProcess("gotmpls", module, {
				args: ["serve-lsp"],
				env: {
					GOTMPLS_DEBUG: "1",
					RUST_BACKTRACE: "1",
					RUST_LOG: "debug",
				},
				stdio: {
					in: this.inputPty.stdio.in, // Client -> Server
					out: this.outputPty.stdio.out, // Server -> Client
					err: this.debugPty.stdio.out, // Server ERR -> Debug PTY
				},
			});
			this.log("✅ WASI process created successfully");

			// Set up reader and writer with their respective PTYs
			this.writer.setPty(this.inputPty); // Client writes to input PTY
			this.reader.setPty(this.outputPty); // Client reads from output PTY
			this.log("✅ PTY communication channels configured");

			// Show the debug terminal
			debugTerminal.show();
			this.log("✅ Debug terminal shown");

			// Run the process in the background
			this.log("▶️  Starting WASI process...");
			try {
				const processPromise = process.run();
				this.log("✅ WASI process started");

				// wait for initialization
				await new Promise((resolve) => setTimeout(resolve, 5000));

				this.initialized = true;
				this.log("🎉 WASI module fully initialized");

				// Handle process completion
				processPromise.then(
					(result) => {
						this.log(`✅ Process completed with code: ${result}`);
						if (result !== 0) {
							this.log(`⚠️  Process exited with non-zero code: ${result}`);
						}
					},
					(error) => {
						this.log(`❌ Process failed with error: ${error}`);
						if (error instanceof Error) {
							this.log(`Stack trace: ${error.stack}`);
						}
					},
				);
			} catch (err) {
				this.log(`❌ Error running WASI process: ${err}`);
				if (err instanceof Error) {
					this.log(`Stack trace: ${err.stack}`);
				}
				throw err;
			}
		} catch (err) {
			this.log(`❌ Error initializing WASI: ${err}`);
			if (err instanceof Error) {
				this.log(`Stack trace: ${err.stack}`);
			}
			throw err;
		}
	}

	override async getVersion(context: vscode.ExtensionContext): Promise<string> {
		// WASI version is tied to extension version
		const extension = vscode.extensions.getExtension("walteh.gotmpls");
		return extension?.packageJSON.version || "unknown";
	}
}
