import * as vscode from 'vscode';
import { spawn } from 'child_process';

let diagnosticCollection: vscode.DiagnosticCollection;

export function activate(context: vscode.ExtensionContext) {
	console.log('Go Template Type Checker is now active');

	// Create diagnostic collection
	diagnosticCollection = vscode.languages.createDiagnosticCollection('go-template');
	context.subscriptions.push(diagnosticCollection);

	// Register handlers
	context.subscriptions.push(
		vscode.workspace.onDidOpenTextDocument(runDiagnostics),
		vscode.workspace.onDidSaveTextDocument(runDiagnostics),
		vscode.workspace.onDidChangeTextDocument((e) => {
			// Debounce diagnostics on change
			const document = e.document;
			if (document.languageId === 'go-template') {
				if (diagnosticsTimeout) {
					clearTimeout(diagnosticsTimeout);
				}
				diagnosticsTimeout = setTimeout(() => runDiagnostics(document), 500);
			}
		})
	);

	// Register hover provider
	context.subscriptions.push(
		vscode.languages.registerHoverProvider('go-template', {
			provideHover(document, position, token) {
				return getHoverInfo(document, position);
			}
		})
	);

	// Register completion provider
	context.subscriptions.push(
		vscode.languages.registerCompletionItemProvider('go-template', {
			async provideCompletionItems(document: vscode.TextDocument, position: vscode.Position, token: vscode.CancellationToken) {
				return getCompletions(document, position);
			}
		})
	);

	// Run diagnostics on all open documents
	vscode.workspace.textDocuments.forEach(runDiagnostics);
}

let diagnosticsTimeout: NodeJS.Timeout | undefined;

async function runDiagnostics(document: vscode.TextDocument) {
	if (document.languageId !== 'go-template') {
		return;
	}

	const config = vscode.workspace.getConfiguration('goTemplateTypes');
	const executable = config.get<string>('executable') || 'go-tmpl-typer';

	const diagnostics: vscode.Diagnostic[] = [];

	try {
		const process = spawn(executable, ['get-diagnostics'], {
			cwd: vscode.workspace.getWorkspaceFolder(document.uri)?.uri.fsPath
		});

		let stdout = '';
		let stderr = '';

		process.stdout.on('data', (data) => {
			stdout += data;
		});

		process.stderr.on('data', (data) => {
			stderr += data;
		});

		await new Promise<void>((resolve, reject) => {
			process.on('close', (code) => {
				if (code === 0) {
					try {
						const results = JSON.parse(stdout);
						for (const diag of results) {
							const range = new vscode.Range(
								diag.start.line - 1,
								diag.start.character - 1,
								diag.end.line - 1,
								diag.end.character - 1
							);
							const diagnostic = new vscode.Diagnostic(
								range,
								diag.message,
								diag.severity === 'error' 
									? vscode.DiagnosticSeverity.Error 
									: vscode.DiagnosticSeverity.Warning
							);
							diagnostics.push(diagnostic);
						}
						resolve();
					} catch (err) {
						reject(new Error('Failed to parse diagnostics output'));
					}
				} else {
					reject(new Error(`Process exited with code ${code}: ${stderr}`));
				}
			});
		});
	} catch (err: any) {
		console.error('Error running diagnostics:', err);
		// Show error in output channel or status bar
		vscode.window.showErrorMessage(`Error running template type checker: ${err.message}`);
	}

	diagnosticCollection.set(document.uri, diagnostics);
}

async function getCompletions(document: vscode.TextDocument, position: vscode.Position): Promise<vscode.CompletionList | undefined> {
	const config = vscode.workspace.getConfiguration('goTemplateTypes');
	const executable = config.get<string>('executable') || 'go-tmpl-typer';

	try {
		const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri)?.uri.fsPath;
		const process = spawn(executable, [
			'get-completions',
			workspaceFolder || '.',
			document.uri.fsPath,
			(position.line + 1).toString(),
			(position.character + 1).toString()
		], {
			cwd: workspaceFolder
		});

		let stdout = '';
		let stderr = '';

		process.stdout.on('data', (data) => {
			stdout += data;
		});

		process.stderr.on('data', (data) => {
			stderr += data;
		});

		const completionItems = await new Promise<vscode.CompletionItem[]>((resolve, reject) => {
			process.on('close', (code) => {
				if (code === 0) {
					try {
						const results = JSON.parse(stdout);
						const items = results.map((item: any) => {
							const completionItem = new vscode.CompletionItem(item.label);
							completionItem.kind = getCompletionKind(item.kind);
							if (item.detail) completionItem.detail = item.detail;
							if (item.documentation) completionItem.documentation = item.documentation;
							if (item.sortText) completionItem.sortText = item.sortText;
							if (item.filterText) completionItem.filterText = item.filterText;
							if (item.insertText) completionItem.insertText = new vscode.SnippetString(item.insertText);
							if (item.textEdit) {
								completionItem.textEdit = new vscode.TextEdit(
									new vscode.Range(
										item.textEdit.range.start.line,
										item.textEdit.range.start.character,
										item.textEdit.range.end.line,
										item.textEdit.range.end.character
									),
									item.textEdit.newText
								);
							}
							return completionItem;
						});
						resolve(items);
					} catch (err) {
						reject(new Error('Failed to parse completions output'));
					}
				} else {
					reject(new Error(`Process exited with code ${code}: ${stderr}`));
				}
			});
		});

		return new vscode.CompletionList(completionItems, false);
	} catch (err: any) {
		console.error('Error getting completions:', err);
		return undefined;
	}
}

function getCompletionKind(kind: string): vscode.CompletionItemKind {
	switch (kind) {
		case 'keyword':
			return vscode.CompletionItemKind.Keyword;
		case 'function':
			return vscode.CompletionItemKind.Function;
		case 'variable':
			return vscode.CompletionItemKind.Variable;
		case 'field':
			return vscode.CompletionItemKind.Field;
		case 'method':
			return vscode.CompletionItemKind.Method;
		default:
			return vscode.CompletionItemKind.Text;
	}
}

async function getHoverInfo(document: vscode.TextDocument, position: vscode.Position): Promise<vscode.Hover | undefined> {
	// TODO: Implement hover info once the Go executable supports it
	// For now, we'll return the type information from diagnostics if available
	const diagnostics = diagnosticCollection.get(document.uri);
	if (!diagnostics) {
		return undefined;
	}

	// Find any diagnostics that overlap with the current position
	const diagnostic = diagnostics.find(d => d.range.contains(position));
	if (diagnostic) {
		return new vscode.Hover(diagnostic.message);
	}

	return undefined;
}

export function deactivate() {
	if (diagnosticCollection) {
		diagnosticCollection.dispose();
	}
} 