import {
    Column,
    Entity,
    JoinColumn,
    OneToMany,
    OneToOne,
    PrimaryColumn,
    PrimaryGeneratedColumn,
} from 'typeorm';
import { Contact } from './contact';
import { MemberInformation } from './memberinformation';

@Entity()
export class Person {
    @PrimaryGeneratedColumn()
    id?: number;

    @PrimaryColumn({ type: 'varchar', length: 30, nullable: false })
    email: string;

    @Column({ type: 'varchar', length: 20, nullable: true })
    firstName?: string;

    @Column({ type: 'varchar', length: 100, nullable: true })
    lastName?: string;

    @Column({ type: 'date', nullable: true })
    birthday?: Date;

    @Column({ type: 'string', length: 10, nullable: true })
    gender?: string;

    @Column({ type: 'blob', nullable: true })
    photo?: Buffer;

    @OneToMany(() => Contact, (contact) => contact.person, { cascade: true })
    contacts?: Contact[];

    @OneToOne(
        () => MemberInformation,
        (memberInformation) => memberInformation.person,
        { cascade: true }
    )
    @JoinColumn()
    memberInformation?: MemberInformation;

    constructor(email: string) {
        this.email = email;
    }
}
