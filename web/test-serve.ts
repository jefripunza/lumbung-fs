import { sql, serve } from "bun";

const server = serve({
  port: 3000,
  routes: {
    "/": new Response("Welcome to Bun!"),
    "/api/validate": async (req) => {
      console.log({ req });
      try {
        const users = await sql`SELECT * FROM users LIMIT 10`;
      return Response.json({ users });
      } catch (error) {
        return Response.json({ error: error }, { status: 500 });
      }
    },
  },
});

console.log(`Listening on localhost:${server.port}`);
