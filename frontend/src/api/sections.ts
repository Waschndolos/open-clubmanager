import {ClubSection} from "./types";

const BASE_URL = "http://localhost:3001/api/sections";

export async function fetchSections(): Promise<ClubSection[]> {
    const res = await fetch(BASE_URL);
    if (!res.ok) throw new Error("Error fetching ClubSection.");
    return res.json();
}

export async function createSection(data: Omit<ClubSection, "id">): Promise<ClubSection> {
    const res = await fetch(BASE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating ClubSection.");
    return res.json();
}


export async function deleteSection(data: ClubSection): Promise<void> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error fetching role.");
    return;
}


export async function updateSection(data: ClubSection): Promise<ClubSection> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating role.");
    return res.json();
}