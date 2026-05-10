import { DataClient } from './dataClient';
import { httpApiClient } from './httpApiClient';

/**
 * Returns the DataClient implementation backed by the REST API.
 */
export function getDataClient(): DataClient {
    return httpApiClient;
}
