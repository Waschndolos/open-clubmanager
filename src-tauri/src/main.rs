#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri_plugin_shell::ShellExt;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            let command = app
                .shell()
                .sidecar("clubmanager-backend")
                .map_err(|error| {
                    std::io::Error::other(format!(
                        "backend sidecar binary is not available: {error}"
                    ))
                })?;

            command.spawn().map_err(|error| {
                std::io::Error::other(format!("failed to start backend sidecar: {error}"))
            })?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running Open ClubManager");
}
