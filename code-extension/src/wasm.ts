/**
 * WASM Engine Implementation
 *
 * This implementation uses WebAssembly to run the language server directly in VS Code.
 * The WASM binary will be compiled from the Go code and loaded at runtime.
 *
 * Architecture:
 * ```
 *  +----------------+         +-----------------+
 *  |   VS Code      |         |    gotmpls     |
 *  |  Extension     |<------->|     WASM       |
 *  +----------------+         +-----------------+
 *                   (in-memory transport)
 * ```
 *
 * TODO:
 * - [ ] Set up WASM build pipeline for gotmpls
 * - [ ] Implement WASM loading and initialization
 * - [ ] Create WASM-based LSP server implementation
 * - [ ] Add memory management and cleanup
 * - [ ] Implement performance monitoring
 */

import * as fs from "fs";
import * as path from "path";

import * as vscode from "vscode";

import {
	DataCallback,
	Disposable,
	Event,
	Message,
	MessageReader,
	MessageWriter,
	PartialMessageInfo,
	StreamInfo,
} from "vscode-languageclient/node";

import { BaseGotmplsEngine, GotmplsEngineType } from "@src/engine";

// Type definitions for callbacks
type ErrorCallback = (error: [Error, Message | undefined, number | undefined]) => void;
type CloseCallback = () => void;

// WASM module interface
declare global {
	interface result {
		result: string;
		error: string | undefined;
	}

	interface GotmplsWasm {
		serve_lsp: (send_message: (msg: string) => void) => void;
	}

	interface Go {
		importObject: WebAssembly.Imports & {
			gojs: {
				"syscall/js.finalizeRef": (v_ref: any) => void;
			};
		};
		run: (instance: any) => void;
	}

	var gotmpls_wasm: GotmplsWasm;
	var gotmpls_initialized: boolean;
	var gotmpls_receive: (msg: string) => void;
}

/**
 * Custom message reader for WASM communication
 */
class WasmMessageReader implements MessageReader {
	private readonly emitter = new vscode.EventEmitter<Message>();
	private connection: MessageConnection | undefined;

	constructor() {
		// Initialize
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
		this.connection = undefined;
	}

	// Method to receive messages from the WASM module
	public receive(message: Message): void {
		if (message) {
			this.emitter.fire(message);
		}
	}
}

/**
 * Custom message writer for WASM communication
 */
class WasmMessageWriter implements MessageWriter {
	private readonly errorEmitter = new vscode.EventEmitter<[Error, Message | undefined, number | undefined]>();
	private readonly closeEmitter = new vscode.EventEmitter<void>();
	private connection: MessageConnection | undefined;
	private initialized = false;

	constructor() {
		// Initialize
	}

	public get onError(): Event<[Error, Message | undefined, number | undefined]> {
		return this.errorEmitter.event;
	}

	public get onClose(): Event<void> {
		return this.closeEmitter.event;
	}

	public async write(msg: Message): Promise<void> {
		if (!this.connection) {
			return Promise.reject(new Error("No connection"));
		}

		try {
			// Convert the message to a string
			const msgStr = JSON.stringify(msg);

			// Send to WASM
			if (globalThis.gotmpls_wasm?.serve_lsp) {
				// First time initialization
				if (!this.initialized) {
					this.initialized = true;
					globalThis.gotmpls_wasm.serve_lsp((response: string) => {
						// Handle response from WASM
						if (this.connection) {
							this.connection.receiveMessage(JSON.parse(response));
						}
					});
				}

				// Send the message through the registered receive function
				if (globalThis.gotmpls_receive) {
					globalThis.gotmpls_receive(msgStr);
				} else {
					throw new Error("WASM receive function not registered");
				}

				return Promise.resolve();
			}

			const error = new Error("WASM module not initialized");
			this.errorEmitter.fire([error, msg, undefined]);
			return Promise.reject(error);
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
		this.connection = undefined;
	}

	public listen(connection: MessageConnection): void {
		this.connection = connection;
	}
}

/**
 * Message connection for coordinating reader and writer
 */
interface MessageConnection {
	receiveMessage(message: any): void;
	sendMessage(message: any): void;
}

export class WasmEngine extends BaseGotmplsEngine {
	private go: Go | null = null;
	private reader: WasmMessageReader;
	private writer: WasmMessageWriter;
	private connection: MessageConnection;

	constructor(outputChannel: vscode.OutputChannel) {
		super(GotmplsEngineType.WASM);
		this.outputChannel = outputChannel;

		// Create reader and writer
		this.reader = new WasmMessageReader();
		this.writer = new WasmMessageWriter();

		// Create connection
		this.connection = {
			receiveMessage: (message: any) => {
				this.reader.receive(message);
			},
			sendMessage: (message: any) => {
				this.writer.write(message);
			},
		};

		// Connect writer
		this.writer.listen(this.connection);
	}

	async createTransport(context: vscode.ExtensionContext): Promise<StreamInfo> {
		return {
			reader: this.reader,
			writer: this.writer,
		};
	}

	private async waitForInit(timeout: number = 5000): Promise<void> {
		this.log("Waiting for WASM initialization...");
		const start = Date.now();
		while (!globalThis.gotmpls_initialized) {
			if (Date.now() - start > timeout) {
				throw new Error("Timeout waiting for WASM initialization");
			}
			await new Promise((resolve) => setTimeout(resolve, 100));
		}
		this.log("WASM initialization complete");
	}

	async initialize(context: vscode.ExtensionContext): Promise<void> {
		if (this.initialized) {
			return;
		}

		this.log("Initializing WASM module...");
		try {
			// Load and execute wasm_exec.js
			let wasmExecPath = path.join(context.extensionPath, "out", "wasm_exec.js");

			const wasmExecPathTinygo = path.join(context.extensionPath, "out", "wasm_exec.tinygo.js");

			let useTinygo = false;

			// check if wasm_exec.golang.js exists
			if (fs.existsSync(wasmExecPathTinygo)) {
				wasmExecPath = wasmExecPathTinygo;
				useTinygo = true;
			}

			const wasmExecContent = await vscode.workspace.fs.readFile(vscode.Uri.file(wasmExecPath));
			let wasmExecContentString = wasmExecContent.toString();

			if (useTinygo) {
				// prevents an error when .String() is called, but does not fully solve the memory leak issue
				// - however, the memory leak is not that bad - https://github.com/tinygo-org/tinygo/issues/1140#issuecomment-1314608377
				wasmExecContentString = wasmExecContentString.replace(
					'"syscall/js.finalizeRef":',
					`"syscall/js.finalizeRef": (v_ref) => {
						const id = mem().getUint32(unboxValue(v_ref), true);
						this._goRefCounts[id]--;
						if (this._goRefCounts[id] === 0) {
							const v = this._values[id];
							this._values[id] = null;
							this._ids.delete(v);
							this._idPool.push(id);
						}
					},
					"syscall/js.finalizeRef-tinygo":`,
				);
			}

			// Create a new context for the WASM execution
			this.log("Creating Go runtime...");
			this.go = new (Function(`
				${wasmExecContentString}
				return Go;
			`)())();

			if (!this.go) {
				throw new Error("Failed to create Go runtime");
			}

			// Load and instantiate the WASM module
			const wasmPath = path.join(context.extensionPath, "out", "gotmpls.wasm");
			this.log(`Loading WASM module from ${wasmPath}`);

			const wasmBuffer = await vscode.workspace.fs.readFile(vscode.Uri.file(wasmPath));
			this.log(`WASM module loaded, size: ${wasmBuffer.length} bytes`);

			const wasmModule = await WebAssembly.compile(wasmBuffer);
			this.log("WASM module compiled");

			const instance = await WebAssembly.instantiate(wasmModule, this.go.importObject);
			this.log("WASM module instantiated");

			this.go.run(instance);
			this.log("WASM module started");

			// Wait for initialization to complete
			await this.waitForInit();
			this.log("WASM module fully initialized");
			this.initialized = true;
		} catch (err) {
			this.log(`Error initializing WASM: ${err}`);
			throw err;
		}
	}

	override async getVersion(context: vscode.ExtensionContext): Promise<string> {
		// WASM version is tied to extension version
		const extension = vscode.extensions.getExtension("walteh.gotmpls");
		return extension?.packageJSON.version || "unknown";
	}
}
