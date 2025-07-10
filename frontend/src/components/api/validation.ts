const BASE_URL = "http://localhost:3001/api/validation";

export async function validatePath(path: string): Promise<{valid: boolean, error?: string}> {
    const res = await fetch(BASE_URL + "/check-db-path",  {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ path }),
    });
    return await res.json();
}