const BASE_URL = "http://localhost:3001/api/settings";

export async function saveDbPath(dbPath: string) {
    const res = await fetch(`${BASE_URL}/set-db-path`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ dbPath }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    console.log("Database path saved:", data);
}