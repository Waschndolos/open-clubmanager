const BASE_URL = `http://localhost:3001/api/auth`;

export async function login(email: string, password: string) {
    const res = await fetch(`${BASE_URL}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
        credentials: "include", // wichtig für httpOnly cookie (Refresh Token)
    });

    if (!res.ok) throw new Error("Login failed");

    return res.json(); // { accessToken, refreshToken (optional) }
}

export async function refreshAccessToken() {
    const res = await fetch(`${BASE_URL}/refresh-token`, {
        method: "POST",
        credentials: "include", // Cookie schicken
    });

    if (!res.ok) throw new Error("Refresh token invalid");

    return res.json(); // { accessToken }
}

export async function getProfile(token: string) {
    const res = await fetch(`${BASE_URL}/profile`, {
        headers: { Authorization: `Bearer ${token}` },
    });

    if (!res.ok) throw new Error("Unauthorized");

    return res.json(); // { email }
}
