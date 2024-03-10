import { Component } from '@angular/core';
import { ThemeService } from '../../service/theme.service';
import { FormsModule } from '@angular/forms';
import { NgForOf } from '@angular/common';
import { SharedModule } from '../../../shared/shared.module';

@Component({
    selector: 'app-header',
    standalone: true,
    imports: [FormsModule, NgForOf, SharedModule],
    templateUrl: './header.component.html',
    styleUrl: './header.component.css',
})
export class HeaderComponent {
    constructor(private themeService: ThemeService) {}

    toggleTheme(): void {
        this.themeService.toggleDarkMode();
    }
}
