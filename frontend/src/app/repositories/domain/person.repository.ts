import { Repository } from 'typeorm';

export class PersonRepository extends Repository<Person> {
    findByEmail(email: string): Promise<Person | null> {
        return this.findOne({ where: { email: email } });
    }
}
