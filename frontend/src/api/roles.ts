import {Group, Role} from "./types";
import {BACKEND_URL} from "./api";

const BASE_URL = `${BACKEND_URL}/roles`;

export async function fetchRoles(): Promise<Role[]> {
    const res = await fetch(BASE_URL);
    if (!res.ok) throw new Error("Error fetching groups.");
    return res.json();
}

export async function createRole(data: Omit<Role, "id">): Promise<Role> {
    const res = await fetch(BASE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating group.");
    return res.json();
}


export async function deleteRole(data: Role): Promise<void> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error fetching role.");
    return;
}


export async function updateRole(data: Role): Promise<Group> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating role.");
    return res.json();
}