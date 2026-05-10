
import "dotenv/config";
import { defineConfig, env } from "prisma/config";

export default defineConfig({
    schema: "./prisma/schema.sqlite.prisma",
    migrations: {
      path: "./prisma/migrations",
    },
    datasource: {
        url: env("DATABASE_URL")
    },
});