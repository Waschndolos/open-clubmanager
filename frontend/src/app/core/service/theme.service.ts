import { Injectable } from '@angular/core';

@Injectable({
    providedIn: 'root',
})
export class ThemeService {
    darkMode: boolean = false;
    constructor() {}

    toggleDarkMode(): void {
        this.darkMode = !this.darkMode;

        if (this.darkMode) {
            document.body.classList.remove('light-theme');
            document.body.classList.add('dark-theme');
        } else {
            document.body.classList.remove('dark-theme');
            document.body.classList.add('light-theme');
        }
    }
}
