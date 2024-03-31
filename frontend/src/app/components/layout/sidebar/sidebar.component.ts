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
        { displayName: $localize`:@@menu_side_home:Home`, route: '/' },
        {
            displayName: $localize`:@@menu_side_members:Members`,
            children: [
                {
                    displayName: $localize`:@@menu_side_member_overview:Member overview`,
                    route: '/members',
                },
                {
                    displayName: $localize`:@@menu_side_member_list:Member list`,
                    route: '/members/list',
                },
            ],
        },
        {
            displayName: $localize`:@@menu_side_contact:Contact`,
            route: '/contact',
        },
    ];

    activeItem: NavItem | null = null;

    setActiveItem(item: NavItem | null, event?: Event) {
        if (event) {
            event.stopPropagation();
        }
        if (item && item.children && item.children.length) {
            item.isOpen = !item.isOpen;
        } else {
            this.activeItem = item;
        }
    }
}
