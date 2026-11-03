// The lizard-script extension: the language (its grammar, brackets and
// snippets are in package.json), the client of the language server that
// `lizardscript lsp` starts, and commands to run, format and check code.
import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;
let terminal: vscode.Terminal | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  context.subscriptions.push(
    vscode.commands.registerCommand("lizard.run", () => runFile(false)),
    vscode.commands.registerCommand("lizard.runVM", () => runFile(true)),
    vscode.commands.registerCommand("lizard.format", () =>
      vscode.commands.executeCommand("editor.action.formatDocument"),
    ),
    vscode.commands.registerCommand("lizard.checkWorkspace", checkWorkspace),
    vscode.commands.registerCommand("lizard.restartServer", restartServer),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("lizard.path")) {
        restartServer();
      }
    }),
    vscode.window.onDidCloseTerminal((closed) => {
      if (closed === terminal) {
        terminal = undefined;
      }
    }),
  );
  await startServer();
}

export async function deactivate(): Promise<void> {
  await client?.stop();
  client = undefined;
}

// lizardscript is the command of the setting lizard.path.
function lizardscript(): string {
  return vscode.workspace.getConfiguration("lizard").get<string>("path") || "lizardscript";
}

async function startServer(): Promise<void> {
  const command = lizardscript();
  const serverOptions: ServerOptions = { command, args: ["lsp"] };
  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: "file", language: "lizard" },
      { scheme: "untitled", language: "lizard" },
    ],
    synchronize: {
      // A module changed on disk changes what the files that use it mean.
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.liz"),
    },
  };
  client = new LanguageClient("lizard", "lizard-script", serverOptions, clientOptions);
  try {
    await client.start();
  } catch (err) {
    client = undefined;
    const choice = await vscode.window.showErrorMessage(
      `lizard-script: can't start \`${command} lsp\` (${err}). Install lizardscript, or set lizard.path to its full path.`,
      "Open settings",
    );
    if (choice === "Open settings") {
      vscode.commands.executeCommand("workbench.action.openSettings", "lizard.path");
    }
  }
}

async function restartServer(): Promise<void> {
  await client?.stop();
  client = undefined;
  await startServer();
}

// quote writes an argument for the terminal's shell.
function quote(arg: string): string {
  if (process.platform === "win32") {
    return `"${arg.replace(/"/g, '""')}"`;
  }
  return `'${arg.replace(/'/g, `'\\''`)}'`;
}

// runFile saves the file of the editor and runs it in a terminal:
// lizardscript file, or lizardscript --vm file.
async function runFile(onVM: boolean): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "lizard") {
    vscode.window.showWarningMessage("lizard-script: open a .liz file to run it.");
    return;
  }
  if (editor.document.isUntitled) {
    vscode.window.showWarningMessage("lizard-script: save the file first, so it has a folder for its modules.");
    return;
  }
  await editor.document.save();
  if (!terminal) {
    terminal = vscode.window.createTerminal("lizard-script");
  }
  terminal.show(true);
  const call = process.platform === "win32" ? "& " : "";
  const args = (onVM ? ["--vm"] : []).concat(editor.document.uri.fsPath).map(quote).join(" ");
  terminal.sendText(`${call}${quote(lizardscript())} ${args}`);
}

// checkWorkspace runs lizardscript check on each folder of the workspace,
// as a task whose findings go to the Problems panel.
async function checkWorkspace(): Promise<void> {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders) {
    vscode.window.showWarningMessage("lizard-script: open a folder to check it.");
    return;
  }
  for (const folder of folders) {
    const task = new vscode.Task(
      { type: "lizard", task: "check" },
      folder,
      "check",
      "lizard-script",
      new vscode.ProcessExecution(lizardscript(), ["check", "."], { cwd: folder.uri.fsPath }),
      "$lizard",
    );
    task.presentationOptions = { reveal: vscode.TaskRevealKind.Silent, clear: true };
    await vscode.tasks.executeTask(task);
  }
}
