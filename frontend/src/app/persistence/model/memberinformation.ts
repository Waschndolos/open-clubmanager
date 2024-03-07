import {
    Column,
    Entity,
    ManyToMany,
    OneToOne,
    PrimaryGeneratedColumn,
} from 'typeorm';
import { Group } from './group';
import { Person } from './person';
import { Financial } from './financial';

@Entity()
export class MemberInformation {
    @PrimaryGeneratedColumn()
    id?: number;

    @Column({ nullable: false })
    memberNumber?: number;

    @Column({ length: 20, nullable: true })
    memberType?: string;

    @Column({ length: 20, nullable: true })
    memberState?: string;

    @Column({ nullable: true })
    entryDate?: Date;

    @Column({ nullable: true })
    exitDate?: Date;

    @ManyToMany(() => Group, (group) => group.members)
    groups?: Group[];

    @OneToOne(() => Person, (person) => person.memberInformation)
    person?: Person;

    @OneToOne(() => Financial, (financial) => financial.memberInformation)
    financial?: Financial;
}
