import * as vscode from 'vscode';
import { spawn } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind
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
				command: executable,
				args: ['serve-lsp', '--debug'],
				options: {
					cwd: workspaceFolder.uri.fsPath,
					env: {
						...process.env,
						GOTMPL_DEBUG: "1",
					}
				}
			};

			outputChannel.appendLine(`Starting language server with command: ${serverOptions.command} ${serverOptions.args?.join(' ')}`);

			// Client options
			const clientOptions: LanguageClientOptions = {
				documentSelector: [{ scheme: 'file', language: 'go-template' }],
				synchronize: {
					fileEvents: vscode.workspace.createFileSystemWatcher('**/*.tmpl')
				},
				outputChannel: outputChannel,
				traceOutputChannel: outputChannel,
				middleware: {
					handleDiagnostics: (uri, diagnostics, next) => {
						outputChannel.appendLine(`Received ${diagnostics.length} diagnostics for ${uri.fsPath}`);
						next(uri, diagnostics);
					},
					provideHover: async (document, position, token, next) => {
						outputChannel.appendLine(`Hover requested at ${document.uri.fsPath}:${position.line}:${position.character}`);
						const result = await next(document, position, token);
						outputChannel.appendLine(`Hover result: ${JSON.stringify(result)}`);
						return result;
					},
					provideCompletionItem: async (document, position, context, token, next) => {
						outputChannel.appendLine(`Completion requested at ${document.uri.fsPath}:${position.line}:${position.character}`);
						const result = await next(document, position, context, token);
						outputChannel.appendLine(`Completion result: ${JSON.stringify(result)}`);
						return result;
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

			// Start the client
			client.start().then(() => {
				outputChannel.appendLine('Language server started');
			}).catch(err => {
				outputChannel.appendLine(`Error starting language server: ${err.message}`);
				outputChannel.appendLine(err.stack || '');
			});

			// Log client state changes
			client.onDidChangeState(event => {
				outputChannel.appendLine(`Client state changed from ${event.oldState} to ${event.newState}`);
			});
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