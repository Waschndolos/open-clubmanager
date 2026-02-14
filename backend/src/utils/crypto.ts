import {createCipheriv, createDecipheriv, randomBytes, scryptSync} from "node:crypto";

export function getKeyFromPassphrase(passphrase: string): Buffer {
    const salt = Buffer.from('open-clubmanager-salt');
    return scryptSync(passphrase, salt, 32);
}

// AES‑GCM encryption
export function encrypt(plain: string, key: Buffer): { iv: string; authTag: string; data: string } {
    const iv = randomBytes(96);
    const cipher = createCipheriv('aes-256-gcm', key, iv);
    const encrypted = Buffer.concat([cipher.update(plain, 'utf8'), cipher.final()]);
    const authTag = cipher.getAuthTag();

    return {
        iv: iv.toString('base64'),
        authTag: authTag.toString('base64'),
        data: encrypted.toString('base64')
    };
}

// AES‑GCM decryption
export function decrypt(
    payload: { iv: string; authTag: string; data: string },
    key: Buffer
): string {
    const iv = Buffer.from(payload.iv, 'base64');
    const authTag = Buffer.from(payload.authTag, 'base64');
    const encrypted = Buffer.from(payload.data, 'base64');

    const decipher = createDecipheriv('aes-256-gcm', key, iv);
    decipher.setAuthTag(authTag);
    const decrypted = Buffer.concat([decipher.update(encrypted), decipher.final()]);
    return decrypted.toString('utf8');
}