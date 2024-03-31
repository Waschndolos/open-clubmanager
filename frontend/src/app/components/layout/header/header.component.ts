import { Component } from '@angular/core';
import { ThemeService } from '../../../services/persistence/theme.service';

@Component({
    selector: 'app-header',
    standalone: true,
    imports: [],
    templateUrl: './header.component.html',
    styleUrl: './header.component.scss',
})
export class HeaderComponent {
    title = $localize`:@@header_title:Open Clubmanager`;
    constructor(private themeService: ThemeService) {}

    toggleTheme(): void {
        this.themeService.toggleDarkMode();
    }
}
