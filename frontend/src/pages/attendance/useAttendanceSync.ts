import { useEffect, useState } from 'react';
import { flushAttendanceQueue, pendingAttendance } from '../../api/attendance';

const RETRY_MS = 20000;

/**
 * Sends attendance recorded offline as soon as the server is reachable again:
 * on mount, when the browser comes back online and periodically. Returns the
 * number of occurrences still waiting.
 */
export function useAttendanceSync(): number {
    const [pending, setPending] = useState(pendingAttendance);

    useEffect(() => {
        const update = () => setPending(pendingAttendance());
        const flush = () => {
            if (pendingAttendance() > 0) void flushAttendanceQueue().then(update);
        };
        flush();
        const timer = window.setInterval(flush, RETRY_MS);
        window.addEventListener('online', flush);
        window.addEventListener('attendance-queue', update);
        return () => {
            window.clearInterval(timer);
            window.removeEventListener('online', flush);
            window.removeEventListener('attendance-queue', update);
        };
    }, []);

    return pending;
}
