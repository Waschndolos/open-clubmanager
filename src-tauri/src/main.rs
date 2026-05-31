#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri_plugin_shell::ShellExt;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            if let Ok(command) = app.shell().sidecar("clubmanager-backend") {
                let _ = command.spawn();
            }

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running Open ClubManager");
}
