import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { HeaderComponent } from './core/component/header/header.component';
import { SidebarComponent } from './core/component/sidebar/sidebar.component';
import { FooterComponent } from './core/component/footer/footer.component';

@NgModule({
    declarations: [AppComponent],
    imports: [
        BrowserModule,
        AppRoutingModule,
        HeaderComponent,
        SidebarComponent,
        FooterComponent,
    ],
    providers: [],
    bootstrap: [AppComponent],
})
export class AppModule {}
