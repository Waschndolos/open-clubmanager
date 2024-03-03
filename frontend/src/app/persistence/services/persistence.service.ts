import {Injectable} from '@angular/core';
import {PersistenceModule} from "../persistence.module";
import * as sqlite3 from 'sqlite3';

@Injectable({
  providedIn: PersistenceModule
})
export class PersistenceService<T> {
  private db: sqlite3.Database;

  constructor(private tableName: string) {
    this.db = new sqlite3.Database('data.db');
  }

  create(item: T): Promise<void> {
    return new Promise((resolve, reject) => {
      this.db.run(`INSERT INTO ${this.tableName} VALUES (?)`, [JSON.stringify(item)], (err) => {
        if (err) {
          reject(err.message);
        } else {
          resolve();
        }
      });
    });
  }

  read(): Promise<T[]> {
    return new Promise((resolve, reject) => {
      this.db.all(`SELECT * FROM ${this.tableName}`, (err, rows: string[]) => {
        if (err) {
          reject(err.message);
        } else {
          resolve(rows.map(row => JSON.parse(row)));
        }
      });
    });
  }

  update(id: number, item: T): Promise<void> {
    return new Promise((resolve, reject) => {
      this.db.run(`UPDATE ${this.tableName} SET data = ? WHERE id = ?`, [JSON.stringify(item), id], (err) => {
        if (err) {
          reject(err.message);
        } else {
          resolve();
        }
      });
    });
  }

  delete(id: number): Promise<void> {
    return new Promise((resolve, reject) => {
      this.db.run(`DELETE FROM ${this.tableName} WHERE id = ?`, [id], (err) => {
        if (err) {
          reject(err.message);
        } else {
          resolve();
        }
      });
    });
  }
}
