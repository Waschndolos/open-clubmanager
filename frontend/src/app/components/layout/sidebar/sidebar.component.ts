import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgClass, NgForOf, NgIf } from '@angular/common';

interface NavItem {
    displayName: string;
    route?: string;
    children?: NavItem[];
    isOpen?: boolean;
}

@Component({
    selector: 'app-sidebar',
    standalone: true,
    templateUrl: './sidebar.component.html',
    styleUrl: './sidebar.component.scss',
    imports: [RouterLink, NgForOf, NgIf, NgClass],
})
export class SidebarComponent {
    navItems: NavItem[] = [
        { displayName: 'Home', route: '/' },
        {
            displayName: 'Members',
            children: [
                {
                    displayName: 'Member overview',
                    route: '/members',
                },
                { displayName: 'Member List', route: '/members/list' },
            ],
        },
        { displayName: 'Kontakt', route: '/contact' },
    ];

    activeItem: NavItem | null = null;

    setActiveItem(item: NavItem | null, event?: Event) {
        if (event) {
            event.stopPropagation();
        }
        if (item && item.children && item.children.length) {
            item.isOpen = !item.isOpen; // Submenü-Status umschalten
        } else {
            this.activeItem = item; // Setzen Sie das aktive Element, wenn es keine Kinder gibt
        }
    }
}
