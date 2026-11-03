//! The Zed extension of lizard-script: it starts the language server,
//! `lizardscript lsp`. The language itself (its grammar, brackets and
//! queries) is in extension.toml and languages/lizard.

use zed_extension_api::{self as zed, settings::LspSettings, LanguageServerId, Result};

struct LizardExtension;

impl zed::Extension for LizardExtension {
    fn new() -> Self {
        LizardExtension
    }

    /// The command of the server: the binary of the settings
    /// (lsp.lizardscript.binary.path), or lizardscript on the PATH, with
    /// the argument lsp.
    fn language_server_command(
        &mut self,
        id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        let binary = LspSettings::for_worktree(id.as_ref(), worktree)
            .ok()
            .and_then(|settings| settings.binary);
        let command = binary
            .as_ref()
            .and_then(|b| b.path.clone())
            .or_else(|| worktree.which("lizardscript"))
            .ok_or_else(|| {
                "lizardscript is not on the PATH: install it, or set \
                 lsp.lizardscript.binary.path in Zed's settings"
                    .to_string()
            })?;
        let args = binary
            .and_then(|b| b.arguments)
            .unwrap_or_else(|| vec!["lsp".to_string()]);
        Ok(zed::Command {
            command,
            args,
            env: worktree.shell_env(),
        })
    }
}

zed::register_extension!(LizardExtension);
