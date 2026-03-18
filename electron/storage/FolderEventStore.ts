import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';
import * as os from 'os';
import { ClubEvent, ClubMeta, Snapshot, SCHEMA_VERSION } from './types.js';
import { LockManager } from './LockManager.js';

type EntityMap = Record<number, Record<string, unknown>>;
type State = Record<string, EntityMap>;

const SNAPSHOT_INTERVAL = 200; // create snapshot after this many events

export class FolderEventStore {
    private clubFolder: string;
    private eventsDir: string;
    private snapshotsDir: string;
    readonly locks: LockManager;

    constructor(clubFolder: string) {
        this.clubFolder = clubFolder;
        this.eventsDir = path.join(clubFolder, 'events');
        this.snapshotsDir = path.join(clubFolder, 'snapshots');
        this.locks = new LockManager(clubFolder);
    }

    ensureFolderStructure(): void {
        const dirs = [
            this.clubFolder,
            this.eventsDir,
            this.snapshotsDir,
            path.join(this.clubFolder, 'locks'),
        ];
        for (const dir of dirs) {
            if (!fs.existsSync(dir)) {
                fs.mkdirSync(dir, { recursive: true });
            }
        }

        const clubJsonPath = path.join(this.clubFolder, 'club.json');
        if (!fs.existsSync(clubJsonPath)) {
            const meta: ClubMeta = {
                name: 'My Club',
                createdAt: new Date().toISOString(),
            };
            this.atomicWrite(clubJsonPath, JSON.stringify(meta, null, 2));
        }

        const schemaVersionPath = path.join(this.clubFolder, 'schema-version.json');
        if (!fs.existsSync(schemaVersionPath)) {
            this.atomicWrite(
                schemaVersionPath,
                JSON.stringify({ version: SCHEMA_VERSION }, null, 2)
            );
        }
    }

    private atomicWrite(filePath: string, content: string): void {
        const tmpPath = filePath + '.tmp';
        fs.writeFileSync(tmpPath, content, 'utf-8');
        fs.renameSync(tmpPath, filePath);
    }

    appendEvent(
        type: string,
        entityKind: string,
        entityId: number,
        patch: Record<string, unknown>,
        actor?: string
    ): ClubEvent {
        const events = this.readAllEvents();
        const revision = events.length;
        const id = crypto.randomUUID();
        const ts = new Date().toISOString();
        const event: ClubEvent = {
            id,
            ts,
            actor: actor ?? os.userInfo().username,
            type,
            entity: { kind: entityKind, id: entityId },
            baseRevision: revision,
            patch,
        };

        const filename = `${ts.replace(/[:.]/g, '-')}_${id}.json`;
        const filePath = path.join(this.eventsDir, filename);
        this.atomicWrite(filePath, JSON.stringify(event, null, 2));

        if (revision + 1 >= SNAPSHOT_INTERVAL && (revision + 1) % SNAPSHOT_INTERVAL === 0) {
            try {
                this.createSnapshot();
            } catch {
                // non-fatal: snapshots are an optimisation
            }
        }

        return event;
    }

    readAllEvents(): ClubEvent[] {
        if (!fs.existsSync(this.eventsDir)) return [];
        const files = fs
            .readdirSync(this.eventsDir)
            .filter((f) => f.endsWith('.json'))
            .sort();

        const events: ClubEvent[] = [];
        for (const file of files) {
            try {
                const raw = fs.readFileSync(path.join(this.eventsDir, file), 'utf-8');
                events.push(JSON.parse(raw) as ClubEvent);
            } catch {
                // skip corrupted event files
            }
        }
        return events;
    }

    private loadLatestSnapshot(): Snapshot | null {
        if (!fs.existsSync(this.snapshotsDir)) return null;
        const files = fs
            .readdirSync(this.snapshotsDir)
            .filter((f) => f.endsWith('.json'))
            .sort()
            .reverse();

        for (const file of files) {
            try {
                const raw = fs.readFileSync(path.join(this.snapshotsDir, file), 'utf-8');
                return JSON.parse(raw) as Snapshot;
            } catch {
                // skip corrupted snapshot
            }
        }
        return null;
    }

    getState(): State {
        const snapshot = this.loadLatestSnapshot();
        const state: State = snapshot ? { ...snapshot.entities } as State : {};
        const startRevision = snapshot ? snapshot.revision : 0;

        const allEvents = this.readAllEvents();
        const events = allEvents.slice(startRevision);

        for (const event of events) {
            const { kind, id } = event.entity;
            if (!state[kind]) {
                state[kind] = {};
            }
            if (event.type.endsWith('.create')) {
                state[kind][id] = { id, ...event.patch };
            } else if (event.type.endsWith('.update')) {
                state[kind][id] = { ...state[kind][id], ...event.patch };
            } else if (event.type.endsWith('.delete')) {
                delete state[kind][id];
            }
        }

        return state;
    }

    getRevision(): number {
        return this.readAllEvents().length;
    }

    createSnapshot(): void {
        const state = this.getState();
        const revision = this.getRevision();
        const snapshot: Snapshot = {
            revision,
            ts: new Date().toISOString(),
            entities: state as Snapshot['entities'],
        };
        const filename = `snapshot-${String(revision).padStart(8, '0')}.json`;
        const filePath = path.join(this.snapshotsDir, filename);
        this.atomicWrite(filePath, JSON.stringify(snapshot, null, 2));
    }

    generateId(): number {
        // Combine millisecond timestamp with 3 random digits to minimise collision
        // probability while staying within Number.MAX_SAFE_INTEGER (2^53 - 1).
        const rand = Math.floor(Math.random() * 1000);
        return Date.now() * 1000 + rand;
    }
}
