import { Component, OnInit } from '@angular/core';
import { ThemeService } from './services/persistence/theme.service';

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.css'],
})
export class AppComponent implements OnInit {
    title = 'open-clubmanager';
    constructor(private _themeService: ThemeService) {}

    ngOnInit(): void {
        this._themeService.initTheme();
    }
}
