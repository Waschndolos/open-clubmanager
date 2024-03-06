import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import {PersonService} from "./service/person.service";



@NgModule({
  declarations: [],
  exports: [PersonService],
  imports: [
    CommonModule
  ]
})
export class PersistenceModule { }
