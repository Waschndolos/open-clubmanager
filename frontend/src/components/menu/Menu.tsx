import Container from "../container/Container";
import {List} from "@mui/material";
import MenuItem from "./MenuItem";

export default function Menu() {
    return (
        <Container>
            <List>
                <MenuItem label="Home"/>
                <MenuItem label="Members"/>
                <MenuItem label="Finance"/>
            </List>
        </Container>
    )
}