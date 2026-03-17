import { ClubSection } from './types';
import { createEntityApi } from './entityApi';

const api = createEntityApi<ClubSection>('sections');

export const fetchSections = api.fetchAll;
export const createSection = api.create;
export const updateSection = api.update;
export const deleteSection = api.delete;