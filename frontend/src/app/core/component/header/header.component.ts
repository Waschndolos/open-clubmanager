import { Component } from '@angular/core';
import { ThemeService } from '../../service/theme.service';

@Component({
    selector: 'app-header',
    standalone: true,
    imports: [],
    templateUrl: './header.component.html',
    styleUrl: './header.component.css',
})
export class HeaderComponent {
    constructor(private themeService: ThemeService) {}

    toggleTheme(): void {
        this.themeService.toggleDarkMode();
    }
}
