import { Injectable } from '@angular/core';
import { PersistenceModule } from '../persistence.module';
import { PersonRepository } from '../repository/person.repository';
import { Person } from '../model/person';

@Injectable({
    providedIn: PersistenceModule,
})
export class PersonService {
    constructor(private repository: PersonRepository) {}

    getUser(email: string): Promise<Person | null> {
        return this.repository.findByEmail(email);
    }
}
