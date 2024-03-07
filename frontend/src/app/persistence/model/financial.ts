import {
    Column,
    Entity,
    JoinColumn,
    OneToOne,
    PrimaryGeneratedColumn,
} from 'typeorm';
import { MemberInformation } from './memberinformation';

@Entity()
export class Financial {
    @PrimaryGeneratedColumn()
    id?: number;

    @Column({ type: 'decimal', precision: 10, scale: 2, nullable: true })
    membershipFee?: number;

    @Column({ type: 'varchar', length: 20, nullable: true })
    paymentStatus?: string;

    @Column({ type: 'varchar', length: 50, nullable: true })
    paymentInterval?: string;

    @Column({ type: 'decimal', precision: 10, scale: 2, nullable: true })
    donations?: number;

    @OneToOne(() => MemberInformation, { nullable: true })
    @JoinColumn()
    memberInformation?: MemberInformation;
}
