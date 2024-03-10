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
    imports: [RouterLink, NgForOf],
    templateUrl: './sidebar.component.html',
    styleUrl: './sidebar.component.css',
})
export class SidebarComponent implements OnInit {
    activeIndex: number = 0;

    links: Link[] = [];

    ngOnInit(): void {
        this.links = [new Link('Home', '/'), new Link('Members', 'member')];
    }

    updateIndex(index: number): void {
        this.activeIndex = index;
    }
}
