import {Column, Entity, PrimaryColumn, PrimaryGeneratedColumn} from "typeorm";

@Entity()
export class Person {

  @PrimaryGeneratedColumn()
  id?: number;

  @Column({type: 'varchar'})
  firstName?: string;

  @Column({type: 'varchar'})
  lastName?: string;

  @PrimaryColumn({type: 'varchar'})
  email: string;

  constructor(email: string) {
    this.email = email;
  }
}
