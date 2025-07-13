import {BACKEND_URL} from "./api";

const BASE_URL = `${BACKEND_URL}/auth`;


export async function login(email: string, password: string) {
    const res = await fetch(`${BASE_URL}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password })
    });

    if (!res.ok) throw new Error("Login failed");

    return res.json(); // { accessToken, refreshToken }
}

export async function getProfile(token: string) {
    const res = await fetch(`${BASE_URL}/profile`, {
        headers: { Authorization: `Bearer ${token}` }
    });

    if (!res.ok) throw new Error("Unauthorized");

    return res.json(); // { email }
}
