import { Member } from "./types";
import { getDataClient } from "./clientFactory";

export async function fetchMembers(): Promise<Member[]> {
    return getDataClient().members.list();
}

export async function deleteMembers(data: Member[]): Promise<void> {
    return getDataClient().members.delete(data);
}

export async function createMember(data: Omit<Member, "id">): Promise<Member> {
    return getDataClient().members.create(data);
}

export async function updateMember(data: Member): Promise<Member> {
    return getDataClient().members.update(data);
}
