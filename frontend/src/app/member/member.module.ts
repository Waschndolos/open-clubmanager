import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Routes } from '@angular/router';
import { MemberListComponent } from './component/member-list/member-list.component';

const routes: Routes = [
    {
        path: '',
        component: MemberListComponent,
    },
];
@NgModule({
    declarations: [],
    imports: [CommonModule, RouterModule.forChild(routes)],
})
export class MemberModule {}
