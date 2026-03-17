import React from 'react';
import {MemberContainingNamedArtifact} from "../../api/types";
import {Chip} from "@mui/material";

type CellRendererProps = { value: unknown; row?: unknown };

export function DateRenderer(props: CellRendererProps) {
    const value = props.value ? new Date(props.value as string).toLocaleDateString() : '';
    return <span>{value}</span>;
}

export function DefaultRenderer(props: CellRendererProps) {
    return <span>{(props.value as string) ?? ""}</span>;
}

export function MemberContainingNamedArtifactRenderer(props: CellRendererProps) {
    const roles = props.value as MemberContainingNamedArtifact[];

    if (!Array.isArray(roles)) return null;

    return (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
            {roles.map((role) => (
                <Chip
                    key={role.id}
                    label={role.name}
                />
            ))}
        </div>
    );
}