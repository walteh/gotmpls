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
	const outputChannel = vscode.window.createOutputChannel('Go Template Type Checker');
	outputChannel.show();
	outputChannel.appendLine('Go Template Type Checker is now active');

	const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
	if (!workspaceFolder) {
		outputChannel.appendLine('No workspace folder found');
		return;
	}

	outputChannel.appendLine(`Workspace folder: ${workspaceFolder.uri.fsPath}`);

	// Find the executable
	findExecutable('go-tmpl-typer', workspaceFolder.uri.fsPath)
		.then(executable => {
			outputChannel.appendLine(`Found executable: ${executable}`);

			// Server options
			const serverOptions: ServerOptions = {
				run: {
					command: executable,
					args: ['serve-lsp'],
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

			outputChannel.appendLine(`Starting language server with command: ${executable} serve-lsp`);

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

			// Client options
			const clientOptions: LanguageClientOptions = {
				documentSelector: [{ scheme: 'file', language: 'go-template' }],
				synchronize: {
					fileEvents: vscode.workspace.createFileSystemWatcher('**/*.{tmpl,go}'),
					configurationSection: 'goTemplateTypes'
				},
				workspaceFolder: workspaceFolder,
				outputChannel: outputChannel,
				traceOutputChannel: outputChannel,
				revealOutputChannelOn: RevealOutputChannelOn.Never,
				initializationOptions: {
					trace: {
						server: vscode.workspace.getConfiguration('goTemplateTypes').get('trace.server')
					}
				}
			};

			// Create and start the client
			client = new LanguageClient(
				'goTemplateTypeChecker',
				'Go Template Type Checker',
				serverOptions,
				clientOptions
			);

			// Register additional error handlers
			client.onDidChangeState(event => {
				outputChannel.appendLine(`Client state changed from ${event.oldState} to ${event.newState}`);
			});

			client.onNotification('window/logMessage', (params: any) => {
				switch (params.type) {
					case 1: // Error
						outputChannel.appendLine(`[Error] ${params.message}`);
						break;
					case 2: // Warning
						outputChannel.appendLine(`[Warning] ${params.message}`);
						break;
					case 3: // Info
						outputChannel.appendLine(`[Info] ${params.message}`);
						break;
					case 4: // Log
						outputChannel.appendLine(`[Log] ${params.message}`);
						break;
				}
			});

			// Start the client
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
			vscode.window.showErrorMessage(`Error starting template type checker: ${err.message}`);
		});
}

async function findExecutable(name: string, workspaceFolder: string | undefined): Promise<string> {
	const config = vscode.workspace.getConfiguration('goTemplateTypes');
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

	throw new Error(`Executable '${name}' not found in PATH. Please install it with 'go install github.com/walteh/go-tmpl-typer/cmd/go-tmpl-typer@latest'`);
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
} 