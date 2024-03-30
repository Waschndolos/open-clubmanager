import { Component } from '@angular/core';
import packageJson from '../../../../../package.json';

@Component({
    selector: 'app-footer',
    standalone: true,
    imports: [],
    templateUrl: './footer.component.html',
    styleUrl: './footer.component.scss',
})
export class FooterComponent {
    public version: string = packageJson.version;
}
