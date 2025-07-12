import { Member } from "./types";

const BASE_URL = "http://localhost:3001/api/members";

export async function fetchMembers(): Promise<Member[]> {
    const res = await fetch(BASE_URL);
    if (!res.ok) throw new Error("Error fetching members.");
    return res.json();
}

export async function deleteMember(data: Member): Promise<void> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error fetching members.");
    return;
}

export async function createMember(data: Omit<Member, "id">): Promise<Member> {
    const res = await fetch(BASE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating member.");
    return res.json();
}

export async function updateMember(data: Member): Promise<Member> {
    const res = await fetch(BASE_URL + "/" + data.id, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error("Error creating member.");
    return res.json();
}
