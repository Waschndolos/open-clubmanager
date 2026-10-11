import { keyframes } from "@mui/material/styles";

export const fadeUp = keyframes`
    from { opacity: 0; transform: translateY(18px); filter: blur(4px); }
    to { opacity: 1; transform: translateY(0); filter: blur(0); }
`;

export const fadeIn = keyframes`
    from { opacity: 0; }
    to { opacity: 1; }
`;

export const slideInLeft = keyframes`
    from { opacity: 0; transform: translateX(-16px); }
    to { opacity: 1; transform: translateX(0); }
`;

export const kenBurns = keyframes`
    from { transform: scale(1.04) translate3d(0, 0, 0); }
    to { transform: scale(1.16) translate3d(-1.5%, -2%, 0); }
`;

export const driftA = keyframes`
    0% { transform: translate3d(-10%, -8%, 0) scale(1); }
    50% { transform: translate3d(22%, 14%, 0) scale(1.25); }
    100% { transform: translate3d(-10%, -8%, 0) scale(1); }
`;

export const driftB = keyframes`
    0% { transform: translate3d(14%, 10%, 0) scale(1.15); }
    50% { transform: translate3d(-18%, -12%, 0) scale(0.9); }
    100% { transform: translate3d(14%, 10%, 0) scale(1.15); }
`;

export const gridPan = keyframes`
    from { background-position: 0 0; }
    to { background-position: 48px 48px; }
`;

export const shake = keyframes`
    10%, 90% { transform: translateX(-1px); }
    20%, 80% { transform: translateX(3px); }
    30%, 50%, 70% { transform: translateX(-6px); }
    40%, 60% { transform: translateX(6px); }
`;

export const sheen = keyframes`
    from { transform: translateX(-120%) skewX(-20deg); }
    to { transform: translateX(260%) skewX(-20deg); }
`;

export const pulseRing = keyframes`
    0% { transform: scale(0.9); opacity: 0.7; }
    100% { transform: scale(2.2); opacity: 0; }
`;

export const EASE = "cubic-bezier(0.22, 1, 0.36, 1)";

/** Entrance animation with a staggered delay (in seconds). */
export const reveal = (delay: number, anim = fadeUp, duration = 0.8) =>
    `${anim} ${duration}s ${EASE} ${delay}s both`;
