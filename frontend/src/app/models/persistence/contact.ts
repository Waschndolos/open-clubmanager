import { Column, Entity, ManyToOne, PrimaryGeneratedColumn } from 'typeorm';
import { Person } from './person';

@Entity()
export class Contact {
    @PrimaryGeneratedColumn()
    id?: number;

    @Column({ length: 200, nullable: true })
    address?: string;

    @Column({ length: 20, nullable: true })
    phoneNumber?: string;

    @Column({ length: 20, nullable: true })
    mobileNumber?: string;

    @ManyToOne(() => Person, (person) => person.contacts)
    person?: Person;
}
