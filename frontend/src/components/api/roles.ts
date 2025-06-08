import { Role } from "./types";

const BASE_URL = "http://localhost:3001/api/roles";

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
