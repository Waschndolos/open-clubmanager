import { Injectable, OnInit } from '@angular/core';
import { LocalStorageService } from './local-storage.service';

@Injectable({
    providedIn: 'root',
})
export class ThemeService implements OnInit {
    constructor(private localStorageService: LocalStorageService) {}

    ngOnInit(): void {}

    toggleDarkMode(): void {
        if (document.documentElement.getAttribute('data-bs-theme') == 'dark') {
            document.documentElement.setAttribute('data-bs-theme', 'light');
            this.localStorageService.save('theme', 'light');
        } else {
            document.documentElement.setAttribute('data-bs-theme', 'dark');
            this.localStorageService.save('theme', 'dark');
        }
    }

    initTheme() {
        let theme = this.localStorageService.load('theme', 'light');

        document.documentElement.setAttribute('data-bs-theme', theme);
        this.localStorageService.save('theme', theme);
    }
}
