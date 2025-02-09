/* --------------------------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Licensed under the MIT License. See License.txt in the project root for license information.
 * ------------------------------------------------------------------------------------------ */

import * as vscode from "vscode";

import { Readable, type Stdio, Wasm, WasmProcess, Writable } from "@vscode/wasm-wasi/v1";
import {
	Disposable,
	Emitter,
	Event,
	Message,
	MessageTransports,
	RAL,
	ReadableStreamMessageReader,
	WriteableStreamMessageWriter,
} from "vscode-languageclient";

import { WasmMessageReader, WasmMessageWriter } from "./wasm";

class ReadableStreamImpl implements RAL.ReadableStream {
	private readonly errorEmitter: Emitter<[Error, Message | undefined, number | undefined]>;
	private readonly closeEmitter: Emitter<void>;
	private readonly endEmitter: Emitter<void>;

	private readonly readable: Readable;

	constructor(readable: Readable) {
		this.errorEmitter = new Emitter<[Error, Message, number]>();
		this.closeEmitter = new Emitter<void>();
		this.endEmitter = new Emitter<void>();
		this.readable = readable;
	}

	public get onData(): Event<Uint8Array> {
		return this.readable.onData;
	}

	public get onError(): Event<[Error, Message | undefined, number | undefined]> {
		return this.errorEmitter.event;
	}

	public fireError(error: any, message?: Message, count?: number): void {
		this.errorEmitter.fire([error, message, count]);
	}

	public get onClose(): Event<void> {
		return this.closeEmitter.event;
	}

	public fireClose(): void {
		this.closeEmitter.fire(undefined);
	}

	public onEnd(listener: () => void): Disposable {
		return this.endEmitter.event(listener);
	}

	public fireEnd(): void {
		this.endEmitter.fire(undefined);
	}
}

type MessageBufferEncoding = RAL.MessageBufferEncoding;

class WritableStreamImpl implements RAL.WritableStream {
	private readonly errorEmitter: Emitter<[Error, Message | undefined, number | undefined]>;
	private readonly closeEmitter: Emitter<void>;
	private readonly endEmitter: Emitter<void>;

	private readonly writable: Writable;

	constructor(writable: Writable) {
		this.errorEmitter = new Emitter<[Error, Message, number]>();
		this.closeEmitter = new Emitter<void>();
		this.endEmitter = new Emitter<void>();
		this.writable = writable;
	}

	public get onError(): Event<[Error, Message | undefined, number | undefined]> {
		const prevEvent = this.errorEmitter.event;
		return (listener: (e: [Error, Message | undefined, number | undefined]) => any): Disposable => {
			return prevEvent((event) => {
				console.log("onError", event);
				listener(event);
			});
		};
	}

	public fireError(error: any, message?: Message, count?: number): void {
		console.log("fireError", error, message, count);
		this.errorEmitter.fire([error, message, count]);
	}

	public get onClose(): Event<void> {
		const prevEvent = this.closeEmitter.event;
		return (listener: () => any, thisArgs?: any, disposables?: Disposable[]): Disposable => {
			return prevEvent(() => {
				console.log("onClose");
				listener();
			});
		};
	}

	public fireClose(): void {
		console.log("fireClose");
		this.closeEmitter.fire(undefined);
	}

	public onEnd(listener: () => void): Disposable {
		console.log("onEnd, this.endEmitter.event", this.endEmitter.event);
		return this.endEmitter.event(listener);
	}

	public fireEnd(): void {
		console.log("fireEnd");
		this.endEmitter.fire(undefined);
	}

	public write(data: string | Uint8Array, _encoding?: MessageBufferEncoding): Promise<void> {
		if (typeof data === "string") {
			console.log("writez", data);
			return this.writable.write(data, "utf-8");
		} else {
			console.log("writeyu", data);
			return this.writable.write(data);
		}
	}

	public end(): void {}
}

// export function createStdioOptions(): Stdio {
// 	const tmpdirz = tmpdir();
// 	const stdin = path.join(tmpdirz, "stdin");
// 	const stdout = path.join(tmpdirz, "stdout");
// 	const stderr = path.join(tmpdirz, "stderr");
// 	return {
// 		in: {
// 			kind: "file",
// 			path: stdin,
// 		},
// 		out: {
// 			kind: "file",
// 			path: stdout,
// 		},
// 		err: {
// 			kind: "file",
// 			path: stderr,
// 		},
// 	};
// }

export function createStdioOptions(wasm: Wasm): Stdio {
	return {
		in: {
			kind: "pipeIn",
		},
		out: {
			kind: "pipeOut",
		},
		err: {
			kind: "pipeOut",
		},
	};
}

export async function startServer(
	process: WasmProcess,
	readable: Readable | undefined = process.stdout,
	writable: Writable | undefined = process.stdin,
): Promise<MessageTransports> {
	if (readable === undefined || writable === undefined) {
		throw new Error("Process created without streams or no streams provided.");
	}

	const reader = new ReadableStreamImpl(readable);
	const writer = new WritableStreamImpl(writable);

	process.run().then(
		(value) => {
			if (value === 0) {
				reader.fireEnd();
			} else {
				reader.fireError([new Error(`Process exited with code: ${value}`), undefined, undefined]);
			}
		},
		(error) => {
			reader.fireError([error, undefined, undefined]);
		},
	);

	return {
		reader: new ReadableStreamMessageReader(reader),
		writer: new WriteableStreamMessageWriter(writer),
		detached: true,
	};
}

export async function startServerFromWasm(
	process: WasmProcess,
	readable: WasmMessageReader,
	writable: WasmMessageWriter,
): Promise<MessageTransports> {
	if (readable === undefined || writable === undefined) {
		throw new Error("Process created without streams or no streams provided.");
	}

	process.run().then(
		(value) => {
			if (value === 0) {
				readable.dispose();
				writable.dispose();
			} else {
				console.log("process.run().then", value);
				writable.write({
					jsonrpc: "2.0",
				});
			}
		},
		(error) => {
			console.log("process.run().then", error);
			writable.write({
				jsonrpc: "2.0",
			});
		},
	);

	return {
		reader: readable,
		writer: writable,
		detached: true,
	};
}

export function createUriConverters():
	| { code2Protocol: (value: vscode.Uri) => string; protocol2Code: (value: string) => vscode.Uri }
	| undefined {
	const folders = vscode.workspace.workspaceFolders;
	if (folders === undefined || folders.length === 0) {
		return undefined;
	}
	const c2p: Map<string, string> = new Map();
	const p2c: Map<string, string> = new Map();
	if (folders.length === 1) {
		const folder = folders[0];
		c2p.set(folder.uri.toString(), "file:///workspace");
		p2c.set("file:///workspace", folder.uri.toString());
	} else {
		for (const folder of folders) {
			const uri = folder.uri.toString();
			c2p.set(uri, `file:///workspace/${folder.name}`);
			p2c.set(`file:///workspace/${folder.name}`, uri);
		}
	}
	return {
		code2Protocol: (uri: vscode.Uri) => {
			const str = uri.toString();
			for (const key of c2p.keys()) {
				if (str.startsWith(key)) {
					return str.replace(key, c2p.get(key) ?? "");
				}
			}
			return str;
		},
		protocol2Code: (value: string) => {
			for (const key of p2c.keys()) {
				if (value.startsWith(key)) {
					return vscode.Uri.parse(value.replace(key, p2c.get(key) ?? ""));
				}
			}
			return vscode.Uri.parse(value);
		},
	};
}
