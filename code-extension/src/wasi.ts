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

import {
	ProcessOptions,
	StdioPipeInDescriptor,
	StdioPipeOutDescriptor,
	Wasm,
	WasmPseudoterminal,
} from "@vscode/wasm-wasi/v1";
import { LanguageClient, LanguageClientOptions, MessageTransports, ServerOptions } from "vscode-languageclient/node";

import { BaseGotmplsEngine, GotmplsEngineType } from "./engine";
import { startMessageConnection } from "./wasm";
import { createStdioOptions, createUriConverters, startServer, startServerFromWasm } from "@src/wasi-wasm-lsp";

// WASI module interface
declare global {
	interface WasiEnv {
		args: string[];
		env: { [key: string]: string };
		preopens: { [key: string]: string };
	}
}

// /**
//  * Converts a WASI Writable to a Node WritableStream
//  * 📝 TODO: Consider adding error handling and backpressure support
//  */
// class WasiToNodeWritableStream extends NodeWritable {
// 	constructor(wasiWritable: Writable) {
// 		super({
// 			write(chunk: any, encoding: BufferEncoding, callback: (error?: Error | null) => void) {
// 				// chunk is a buffer, convert to utf-8
// 				const utf8Chunk = new TextEncoder().encode(chunk);
// 				console.log("writing to wasi", utf8Chunk);
// 				return wasiWritable.write(utf8Chunk);
// 			},
// 			defaultEncoding: "utf-8",
// 			highWaterMark: 1024,
// 			decodeStrings: false,
// 			emitClose: true,
// 		});
// 	}
// }

// /**
//  * Converts a WASI Readable to a Node ReadableStream
//  * 📝 TODO: Consider adding error handling and buffering support
//  */
// class WasiToNodeReadableStream extends NodeReadable {
// 	constructor(wasiReadable: Readable) {
// 		super({
// 			read(size) {
// 				wasiReadable.onData((chunk) => {
// 					console.log("reading from wasi", new TextDecoder().decode(chunk));
// 					this.push(chunk);
// 				});
// 			},
// 		});
// 	}
// }

export async function wasi_activate(
	context: vscode.ExtensionContext,
	outputChannel: vscode.OutputChannel,
	baseClientOptions: LanguageClientOptions,
) {
	const wasm: Wasm = await Wasm.load();

	const channel = outputChannel;
	// The server options to run the WebAssembly language server.
	const serverOptions: ServerOptions = async () => {
		const [connection, reader, writer] = startMessageConnection();

		const options: ProcessOptions = {
			trace: true,

			env: {
				GOTMPLS_DEBUG: "1",
				RUST_BACKTRACE: "1",
				RUST_LOG: "debug",
				zzz: {
					yo_send: (msg: string) => {
						console.log("yo_send", msg);
					},
					yo_recv: (msg: string) => {
						console.log("yo_recv b", msg);
					},
				},
			},
			stdio: createStdioOptions(wasm),
			mountPoints: [{ kind: "workspaceFolder" }],
		};

		// Load the WebAssembly code
		const filename = vscode.Uri.joinPath(context.extensionUri, "out", "gotmpls.wasi.wasm");
		const bits = await vscode.workspace.fs.readFile(filename);
		const module = await WebAssembly.compile(bits);

		const memory: WebAssembly.MemoryDescriptor | WebAssembly.Memory = {
			initial: 10000,
			maximum: 10000,
			shared: true,
			buffer: new ArrayBuffer(10000),
		};

		// Create the wasm worker that runs the LSP server
		const process = await wasm.createProcess("gotmpls", module, memory, options);

		// Hook stderr to the output channel
		const decoder = new TextDecoder("utf-8");
		process.stderr!.onData((data) => {
			channel.appendLine("[wasi-stderr] " + decoder.decode(data));
		});

		// process.stdout!.onData((data) => {
		// 	channel.appendLine("[wasi-stdout] " + decoder.decode(data));
		// });

		return startServerFromWasm(process, reader, writer);
	};

	baseClientOptions.uriConverters = createUriConverters();

	let client = new LanguageClient("gotmpls", "gotmpls", serverOptions, baseClientOptions);

	await client.start();
}

export class WasiEngine extends BaseGotmplsEngine {
	// private wasi: Wasm | null = null;
	// private reader: Readable | null = null;
	// private writer: Writable | null = null;
	private debugPty: WasmPseudoterminal | undefined; // Keep PTY for debug output

	constructor(outputChannel: vscode.OutputChannel) {
		super(GotmplsEngineType.WASI);
		this.outputChannel = outputChannel;
	}

	async initialize(
		context: vscode.ExtensionContext,
		outputChannel: vscode.OutputChannel,
	): Promise<MessageTransports> {
		// if (this.initialized) {
		// 	return;
		// }

		this.log("🔍 Starting WASI initialization...");
		try {
			// Create WASI instance
			this.log("📦 Creating WASI instance...");
			const wasm = await Wasm.load();
			// this.wasi = wasm;
			this.log("✅ WASI instance created successfully");

			// Create WASI process with PTY stdio
			this.log("🏗️  Creating PTYs...");

			// Create separate PTYs for LSP communication and debug
			this.debugPty = wasm.createPseudoterminal(); // For debug output
			this.log("✅ Created PTYs for LSP communication and debug");

			// Create a terminal for debug output
			this.log("🚀 Creating VS Code debug terminal...");
			const debugTerminal = vscode.window.createTerminal({
				name: "gotmpls LSP Debug",
				pty: this.debugPty as unknown as vscode.Pseudoterminal,
			});

			const stdErr = wasm.createReadable();

			// Add verbose debug output handler
			stdErr.onData((data: Uint8Array<ArrayBufferLike>) => {
				// parse the data as utf-8
				const str = new TextDecoder().decode(data);
				this.log(`[wasi-stderr] ${str}`);
			});

			// Load the WASI module
			const wasmPath = path.join(context.extensionPath, "out", "gotmpls.wasi.wasm");
			this.log(`📂 Loading WASI module from: ${wasmPath}`);
			const wasmBits = await vscode.workspace.fs.readFile(vscode.Uri.file(wasmPath));
			this.log(`📥 WASI module loaded, size: ${wasmBits.length} bytes`);

			this.log("🔧 Compiling WASI module...");
			const module = await WebAssembly.compile(wasmBits);
			this.log("✅ WASI module compiled successfully");

			const writer = wasm.createWritable();
			const reader = wasm.createReadable();

			const writeToServerPipe: StdioPipeInDescriptor = {
				kind: "pipeIn",
				pipe: writer,
			};

			const readFromServerPipe: StdioPipeOutDescriptor = {
				kind: "pipeOut",
				pipe: reader,
			};

			const stdErrPipe: StdioPipeOutDescriptor = {
				kind: "pipeOut",
				pipe: stdErr,
			};

			// Create WASI process with split stdio
			const process = await wasm.createProcess("gotmpls", module, {
				args: ["serve-lsp"],
				env: {
					GOTMPLS_DEBUG: "1",
					RUST_BACKTRACE: "1",
					RUST_LOG: "debug",
				},
				stdio: {
					in: writeToServerPipe, // Client -> Server
					out: readFromServerPipe, // Server -> Client
					err: stdErrPipe, // Server ERR -> Debug PTY
				},
			});
			this.log("✅ WASI process created successfully");

			// Set up reader and writer with their respective PTYs
			// this.writer.setPty(this.debugPty); // Client writes to input PTY
			// this.reader.setPty(this.outputPty); // Client reads from output PTY
			this.log("✅ PTY communication channels configured");

			// Show the debug terminal
			debugTerminal.show();
			this.log("✅ Debug terminal shown");

			return startServer(process, reader, writer);

			// // Run the process in the background
			// this.log("▶️  Starting WASI process...");
			// try {
			// 	const transport = await startServer(process, this.reader!, this.writer!);
			// 	this.log("✅ WASI process started");

			// 	// wait for initialization
			// 	await new Promise((resolve) => setTimeout(resolve, 5000));

			// 	this.initialized = true;
			// 	this.log("🎉 WASI module fully initialized");

			// 	// Handle process completion
			// 	processPromise.then(
			// 		(result) => {
			// 			this.log(`✅ Process completed with code: ${result}`);
			// 			if (result !== 0) {
			// 				this.log(`⚠️  Process exited with non-zero code: ${result}`);
			// 			}
			// 		},
			// 		(error) => {
			// 			this.log(`❌ Process failed with error: ${error}`);
			// 			if (error instanceof Error) {
			// 				this.log(`Stack trace: ${error.stack}`);
			// 			}
			// 		},
			// 	);
			// } catch (err) {
			// 	this.log(`❌ Error running WASI process: ${err}`);
			// 	if (err instanceof Error) {
			// 		this.log(`Stack trace: ${err.stack}`);
			// 	}
			// 	throw err;
			// }
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
