import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { HomeComponent } from './components/domain/home/home.component';
import { MemberListComponent } from './components/domain/members/member-list/member-list.component';
import { MemberOverviewComponent } from './components/domain/members/member-overview/member-overview.component';

const routes: Routes = [
    {
        path: '',
        component: HomeComponent,
    },
    {
        path: 'members/list',
        component: MemberListComponent,
    },
    {
        path: 'members',
        component: MemberOverviewComponent,
    },
];

@NgModule({
    imports: [RouterModule.forRoot(routes)],
    exports: [RouterModule],
})
export class AppRoutingModule {}
