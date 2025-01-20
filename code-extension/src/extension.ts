import * as vscode from 'vscode';
import { spawn } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind,
	RevealOutputChannelOn
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
	const outputChannel = vscode.window.createOutputChannel('gotmpls');
	outputChannel.show();
	outputChannel.appendLine('gotmpls is now active');

	const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
	if (!workspaceFolder) {
		outputChannel.appendLine('No workspace folder found');
		return;
	}

	outputChannel.appendLine(`workspace folder: ${workspaceFolder.uri.fsPath}`);

	// Find the executable
	findExecutable('gotmpls', workspaceFolder.uri.fsPath)
		.then(executable => {
			outputChannel.appendLine(`found executable: ${executable}`);

			// Server options
			const serverOptions: ServerOptions = {
				run: {
					command: executable,
					args: ['serve-lsp', '--debug'],
					options: {
						cwd: workspaceFolder.uri.fsPath,
						env: {
							...process.env,
							GOTMPL_DEBUG: "1",
							GOPATH: process.env.GOPATH || path.join(process.env.HOME || '', 'go'),
							GOMODCACHE: process.env.GOMODCACHE || path.join(process.env.HOME || '', 'go', 'pkg', 'mod'),
							GO111MODULE: "on",
						}
					}
				},
				debug: {
					command: executable,
					args: ['serve-lsp', '--debug'],
					options: {
						cwd: workspaceFolder.uri.fsPath,
						env: {
							...process.env,
							GOTMPL_DEBUG: "1",
							GOPATH: process.env.GOPATH || path.join(process.env.HOME || '', 'go'),
							GOMODCACHE: process.env.GOMODCACHE || path.join(process.env.HOME || '', 'go', 'pkg', 'mod'),
							GO111MODULE: "on",
						}
					}
				}
			};

			outputChannel.appendLine(`starting gotmpls with command: ${executable} serve-lsp`);

			// Create a wrapper for stderr to capture debug output
			const serverProcess = spawn(executable, ['serve-lsp'], {
				cwd: workspaceFolder.uri.fsPath,
				env: {
					...process.env,
					GOTMPL_DEBUG: "1",
					GOPATH: process.env.GOPATH || path.join(process.env.HOME || '', 'go'),
					GOMODCACHE: process.env.GOMODCACHE || path.join(process.env.HOME || '', 'go', 'pkg', 'mod'),
					GO111MODULE: "on",
				}
			});
			serverProcess.stderr.on('data', (data) => {
				outputChannel.appendLine(`[Server Debug] ${data}`);
			});

			serverProcess.on('error', (err) => {
				outputChannel.appendLine(`[Server Error] ${err.message}`);
			});

			// Client options
			const clientOptions: LanguageClientOptions = {
				documentSelector: [{ scheme: 'file', language: 'gotmpl' }],
				synchronize: {
					fileEvents: vscode.workspace.createFileSystemWatcher('**/*.{tmpl,go}'),
					configurationSection: 'gotmpls'
				},
				workspaceFolder: workspaceFolder,
				outputChannel: outputChannel,
				traceOutputChannel: outputChannel,
				revealOutputChannelOn: RevealOutputChannelOn.Never,
				initializationOptions: {
					trace: {
						server: vscode.workspace.getConfiguration('gotmpls').get('trace.server')
					}
				}
			};

			// Create and start the client
			client = new LanguageClient(
				'gotmpls',
				'gotmpls',
				serverOptions,
				clientOptions
			);

			// Register additional error handlers
			client.onDidChangeState(event => {
				outputChannel.appendLine(`Client state changed from ${event.oldState} to ${event.newState}`);
			});

			client.onNotification('telemetry/event', (params: any) => {
				// // Skip debug logs unless in debug mode
				// if (params.type >= 4) return;

				var str = ""
				switch (params.type) {
					case 1: // Error
						str = `🟥 error      `;
						break;
					case 2: // Warning
						str = `🟧 warning    `;
						break;
					case 3: // Info
						str = `🟦 info       `;
						break;
					case 5: // Trace
						str = `🟪 debug      `;
						break;
					case 4: // Debug
						str = `⬜ trace      `;
						break;
					case 6: // Dependency
						str = `⬜ dependency `;
						break;
				}

				// if trace is disabled, skip 4 and 6
				if (!vscode.workspace.getConfiguration('gotmpls').get('trace.server')) {
					if (params.type === 4 || params.type === 6) return;
				}

				// Add time and source if available
				if (params.time) str += `${params.time} `;
				if (params.source) str += `${params.source} `;

				// Add direction and method if available
				// if (params.direction) str += `${params.direction} `;
				// if (params.method) str += `${params.method} `;

				// Add message
				str += `- ${params.message}`;

				// Add extra fields if any
				if (params.extra) {
					const extras = Object.entries(params.extra)
						// .filter(([key]) => !['level', 'direction', 'method'].includes(key)) // These are handled separately
						.map(([key, value]) => `${key}=${value}`)
						.join(' ');
					if (extras) str += ` | ${extras}`;
				}

				outputChannel.appendLine(str);
			});
			client.start().catch(err => {
				outputChannel.appendLine(`Error starting language server: ${err.message}`);
				outputChannel.appendLine(err.stack || '');
			});

			// Register the client for cleanup
			context.subscriptions.push({
				dispose: () => {
					client?.stop();
				}
			});

			outputChannel.appendLine('Language server started');
		})
		.catch(err => {
			outputChannel.appendLine(`Error finding executable: ${err.message}`);
			outputChannel.appendLine(err.stack || '');
			vscode.window.showErrorMessage(`Error starting  gotmpls: ${err.message}`);
		});
}

async function findExecutable(name: string, workspaceFolder: string | undefined): Promise<string> {
	const config = vscode.workspace.getConfiguration('gotmpls');
	let executable = config.get<string>('executable') || name;

	// If it's an absolute path, verify it exists
	if (path.isAbsolute(executable)) {
		if (fs.existsSync(executable)) {
			return executable;
		}
		throw new Error(`Executable not found at configured path: ${executable}`);
	}

	// If it's a relative path and we have a workspace folder, check relative to that
	if (workspaceFolder && !executable.startsWith('./') && !executable.startsWith('../')) {
		const workspacePath = path.join(workspaceFolder, executable);
		if (fs.existsSync(workspacePath)) {
			return workspacePath;
		}
	}

	// Check if it's in PATH
	const envPath = process.env.PATH || '';
	const pathSeparator = process.platform === 'win32' ? ';' : ':';
	const pathDirs = envPath.split(pathSeparator);

	for (const dir of pathDirs) {
		const fullPath = path.join(dir, executable);
		if (fs.existsSync(fullPath)) {
			return fullPath;
		}
	}

	throw new Error(`Executable '${name}' not found in PATH. Please install it with 'go install github.com/walteh/gotmpls/cmd/gotmpls@latest'`);
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
} 

// figurie ity out