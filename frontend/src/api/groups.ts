import { Group } from "./types";
import {BACKEND_URL} from "./api";

const BASE_URL = `${BACKEND_URL}/groups`;

export async function fetchGroups(): Promise<Group[]> {
    const res = await fetch(BASE_URL);
    if (!res.ok) throw new Error("Error fetching groups.");
    return res.json();
}

export async function deleteGroup(data: Group): Promise<void> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "DELETE",
            headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error fetching groups.");
    return;
}

export async function createGroup(data: Omit<Group, "id">): Promise<Group> {
    const res = await fetch(BASE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating group.");
    return res.json();
}

export async function updateGroup(data: Group): Promise<Group> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating group.");
    return res.json();
}
