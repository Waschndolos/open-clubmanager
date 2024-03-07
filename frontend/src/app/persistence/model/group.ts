import {
    Column,
    Entity,
    JoinTable,
    ManyToMany,
    PrimaryGeneratedColumn,
} from 'typeorm';
import { MemberInformation } from './memberinformation';

@Entity()
export class Group {
    @PrimaryGeneratedColumn()
    id?: number;

    @Column({ nullable: false })
    name?: string;

    @ManyToMany(() => MemberInformation, (person) => person.groups)
    @JoinTable()
    members?: MemberInformation[];
}
