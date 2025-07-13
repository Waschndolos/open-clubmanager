import React, { useEffect, useState } from "react";
import { useAuth } from "../../context/AuthContext";
import { getProfile } from "../../api/authentication";

export default function Profile() {
    const { accessToken } = useAuth();
    const [email, setEmail] = useState("");

    useEffect(() => {
        if (accessToken) {
            getProfile(accessToken).then(data => setEmail(data.email));
        }
    }, [accessToken]);

    if (!accessToken) return <p>Bitte einloggen</p>;

    return <h2>Willkommen {email}</h2>;
}
