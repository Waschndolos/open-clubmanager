import api from './api';
import { Member } from "./types";
import {BACKEND_URL} from "./api";

const BASE_PATH = `/members`;

export async function fetchMembers(): Promise<Member[]> {
    const res = await fetch(`${BACKEND_URL}/members`);
    if (!res.ok) throw new Error("Error fetching members.");
    return res.json();
}

export async function deleteMembers(data: Member[]): Promise<void> {
    for (const member of data) {
        await api.delete(`${BASE_PATH}/${member.id}`);
    }
    return;
}

export async function createMember(data: Omit<Member, "id">): Promise<Member> {
    const res = await api.post<Member>(BASE_PATH, data);
    return res.data;
}

export async function updateMember(data: Member): Promise<Member> {
    const res = await api.put<Member>(`${BASE_PATH}/${data.id}`, data);
    return res.data;
}
