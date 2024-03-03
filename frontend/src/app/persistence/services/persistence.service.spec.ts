import { PersistenceService } from './persistence.service';

describe('PersistenceService', () => {
  let repository: PersistenceService<any>;

  beforeEach(() => {
    repository = new PersistenceService<any>('items');
  });

  it('should create a new item', (done) => {
    repository.create({ id: 1, name: 'Test Item' }).then(() => {
      repository.read().then(items => {
        expect(items.length).toBe(1);
        expect(items[0].id).toBe(1);
        expect(items[0].name).toBe('Test Item');
        done();
      });
    });
  });

  it('should update an existing item', (done) => {
    repository.create({ id: 1, name: 'Test Item' }).then(() => {
      repository.update(1, { id: 1, name: 'Updated Item' }).then(() => {
        repository.read().then(items => {
          expect(items.length).toBe(1);
          expect(items[0].id).toBe(1);
          expect(items[0].name).toBe('Updated Item');
          done();
        });
      });
    });
  });

});
