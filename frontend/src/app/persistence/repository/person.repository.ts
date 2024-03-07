import { Repository } from 'typeorm';
import { Person } from '../model/person';

export class PersonRepository extends Repository<Person> {
    findByEmail(email: string): Promise<Person | null> {
        return this.findOne({ where: { email: email } });
    }
}
