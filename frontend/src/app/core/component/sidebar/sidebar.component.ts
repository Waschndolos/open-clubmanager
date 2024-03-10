import { Component, OnInit } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgForOf } from '@angular/common';

export class Link {
    name: string;
    link: string;
    constructor(name: string, link: string) {
        this.name = name;
        this.link = link;
    }
}

@Component({
    selector: 'app-sidebar',
    standalone: true,
    templateUrl: './sidebar.component.html',
    styleUrl: './sidebar.component.css',
    imports: [RouterLink, NgForOf],
})
export class SidebarComponent implements OnInit {
    activeIndex: number = 0;

    links: Link[] = [];

    ngOnInit(): void {
        this.links = [
            new Link($localize`Sidebar.Home`, '/'),
            new Link($localize`Sidebar.Members`, 'member'),
        ];
    }

    updateIndex(index: number): void {
        this.activeIndex = index;
    }
}
