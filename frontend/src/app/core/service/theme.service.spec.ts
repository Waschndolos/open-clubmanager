import { TestBed } from '@angular/core/testing';

import { ThemeService } from './theme.service';

describe('ThemeService', () => {
    let service: ThemeService;

    beforeEach(() => {
        TestBed.configureTestingModule({});
        service = TestBed.inject(ThemeService);
    });

    it('should be created', () => {
        expect(service).toBeTruthy();
    });

    it('should toggle dark mode', () => {
        expect(service.darkMode).toBeFalsy();

        service.toggleDarkMode();
        expect(service.darkMode).toBeTruthy();
        expect(document.body.classList.contains('dark-theme')).toBeTruthy();

        service.toggleDarkMode();
        expect(service.darkMode).toBeFalsy();
        expect(document.body.classList.contains('dark-theme')).toBeFalsy();
        expect(document.body.classList.contains('light-theme')).toBeTruthy();
    });
});
