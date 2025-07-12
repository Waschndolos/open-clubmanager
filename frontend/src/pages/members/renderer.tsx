import React from 'react';
import {ICellRendererParams} from "ag-grid-community";
import {MemberContainingNamedArtifact} from "../../api/types";
import {Chip} from "@mui/material";

export function DateRenderer(props: ICellRendererParams) {
    const value = props.value ? new Date(props.value).toLocaleDateString() : '';
    return <span>{value}</span>;
}

export function DefaultRenderer(props: ICellRendererParams) {
    return <span>{props.value ?? ""}</span>;
}

export function MemberContainingNamedArtifactRenderer(props: ICellRendererParams) {
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