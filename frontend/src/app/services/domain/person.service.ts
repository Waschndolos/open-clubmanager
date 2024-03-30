import { Injectable } from '@angular/core';
import { PersonRepository } from '../../repositories/domain/person.repository';

@Injectable({
    providedIn: 'root',
})
export class PersonService {
    constructor(private repository: PersonRepository) {}

    getUser(email: string): Promise<Person | null> {
        return this.repository.findByEmail(email);
    }
}
