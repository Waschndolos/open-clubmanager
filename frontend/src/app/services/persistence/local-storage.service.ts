import { Injectable } from '@angular/core';

@Injectable({
    providedIn: 'root',
})
export class LocalStorageService {
    constructor() {}

    save(key: string, value: string) {
        localStorage.setItem(key, value);
    }

    load(key: string, defaultValue: string): string {
        let value = localStorage.getItem(key);
        return value != null ? value : defaultValue;
    }
}
